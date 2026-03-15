package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
	"unicode"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(func(r rune) bool {
		return unicode.Is(unicode.Mn, r)
	}))
	result, _, _ := transform.String(t, s)
	return result
}

func formatCityName(s string) string {
	// Remove acentos
	s = removeAccents(s)
	// Substitui espacos por underline
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, "_")
	return s
}

var cepRegex = regexp.MustCompile(`^[0-9]{8}$`)

type requestPayload struct {
	Cep string `json:"cep"`
}

type weatherResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type viaCepResponse struct {
	Localidade string `json:"localidade"`
	Erro       string `json:"erro"`
}

type outputPayload struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func main() {
	ctx := context.Background()

	// Wait for OTEL Collector to be ready
	time.Sleep(2 * time.Second)

	shutdown, err := initTracer(ctx)
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() {
		_ = shutdown(ctx)
	}()

	http.Handle("/zipcode", otelhttp.NewHandler(http.HandlerFunc(zipcodeHandler), "service-b"))

	addr := ":8081"
	log.Printf("service-b listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func initTracer(ctx context.Context) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	var exporter trace.SpanExporter
	var err error

	for i := range 5 {
		exporter, err = otlptracegrpc.New(ctx,
			otlptracegrpc.WithEndpoint(endpoint),
			otlptracegrpc.WithInsecure(),
		)
		if err == nil {
			break
		}
		log.Printf("Failed to create exporter (attempt %d/5): %v", i+1, err)
		time.Sleep(time.Second * 2)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create exporter after retries: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("service-b"),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)
	otel.SetTracerProvider(provider)

	return provider.Shutdown, nil
}

func zipcodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var p requestPayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !cepRegex.MatchString(p.Cep) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		writeJSON(w, map[string]string{"message": "invalid zipcode"})
		return
	}

	city, err := fetchCity(r.Context(), p.Cep)
	if err != nil {
		if err == errZipcodeNotFound {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]string{"message": "can not find zipcode"})
			return
		}
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	celsius, err := fetchTemperature(r.Context(), city)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		writeJSON(w, map[string]string{"message": err.Error()})
		return
	}

	out := outputPayload{
		City:  city,
		TempC: celsius,
		TempF: celsius*1.8 + 32,
		TempK: celsius + 273,
	}
	writeJSON(w, out)
}

var errZipcodeNotFound = fmt.Errorf("zipcode not found")

func fetchCity(ctx context.Context, cep string) (string, error) {
	ctx, span := otel.Tracer("service-b").Start(ctx, "fetch-via-cep")
	defer span.End()

	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	span.SetAttributes(attribute.String("url", url))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport), Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var v viaCepResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "", err
	}

	if v.Erro == "true" || v.Localidade == "" {
		return "", errZipcodeNotFound
	}

	return formatCityName(v.Localidade), nil
}

func fetchTemperature(ctx context.Context, city string) (float64, error) {
	ctx, span := otel.Tracer("service-b").Start(ctx, "fetch-weather")
	defer span.End()

	key := os.Getenv("WEATHERAPI_KEY")
	if key == "" {
		return 0, fmt.Errorf("missing WEATHERAPI_KEY")
	}

	url := fmt.Sprintf("https://api.weatherapi.com/v1/current.json?key=%s&q=%s&aqi=no", key, city)
	span.SetAttributes(attribute.String("url", url), attribute.String("city", city))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport), Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("weather api %d: %s", resp.StatusCode, string(body))
	}

	var w weatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&w); err != nil {
		return 0, err
	}

	return w.Current.TempC, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
