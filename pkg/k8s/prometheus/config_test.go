package prometheus_test

import (
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/apis/indicatordocument/v1"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/k8s/prometheus"
	"github.com/pivotal/monitoring-indicator-protocol/pkg/prometheus_alerts"
)

var indicators = []*v1.IndicatorDocument{
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my_app_indicators",
			Namespace: "monitoring",
			Labels: map[string]string{
				"environment": "staging",
			},
		},
		Spec: v1.IndicatorDocumentSpec{
			Product: v1.Product{
				Name:    "my_app",
				Version: "1.0.1",
			},
			Indicators: []v1.IndicatorSpec{
				{
					Name:   "latency",
					PromQL: "histogram_quantile(0.9, latency)",
					Thresholds: []v1.Threshold{
						{
							Level:    "critical",
							Operator: v1.GreaterThanOrEqualTo,
							Value:    float64(100.2),
							Alert: v1.Alert{
								For:  "5m",
								Step: "10s",
							},
						},
					},
					Documentation: map[string]string{
						"title": "90th Percentile Latency",
					},
				},
			},
			Layout: v1.Layout{},
		},
	},
	{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my_production_app_indicators",
			Namespace: "monitoring",
			Labels: map[string]string{
				"environment": "production",
			},
		},
		Spec: v1.IndicatorDocumentSpec{
			Product: v1.Product{
				Name:    "my_app",
				Version: "1.0.1",
			},
			Indicators: []v1.IndicatorSpec{
				{
					Name:   "average_latency",
					PromQL: "average(latency)",
					Thresholds: []v1.Threshold{
						{
							Level:    "warning",
							Operator: v1.NotEqualTo,
							Value:    float64(0),
							Alert: v1.Alert{
								For:  "10m",
								Step: "10s",
							},
						},
					},
					Documentation: map[string]string{
						"title": "Average Latency",
					},
				},
			},
			Layout: v1.Layout{},
		},
	},
}

func TestConfig(t *testing.T) {
	t.Run("it renders empty groups when there is no indicators", func(t *testing.T) {
		g := NewGomegaWithT(t)
		p := prometheus.NewConfig()

		g.Expect(p.String()).To(MatchYAML("groups: []"))
	})

	t.Run("it renders a group for each indicator document", func(t *testing.T) {
		testCases := map[string]struct {
			Indicators []*v1.IndicatorDocument
			Expected   string
		}{
			"1 document": {
				Indicators: []*v1.IndicatorDocument{indicators[0]},
				Expected: `
                    groups:
                    - name: monitoring/my_app_indicators
                      rules:
                      - alert: latency
                        expr: histogram_quantile(0.9, latency) >= 100.2
                        for: 5m
                        labels:
                          product: my_app
                          version: 1.0.1
                          level: critical
                          environment: staging
                        annotations:
                          title: 90th Percentile Latency
                `,
			},
			"2 documents": {
				Indicators: indicators,
				Expected: `
                    groups:
                    - name: monitoring/my_app_indicators
                      rules:
                      - alert: latency
                        annotations:
                          title: 90th Percentile Latency
                        expr: histogram_quantile(0.9, latency) >= 100.2
                        for: 5m
                        labels:
                          environment: staging
                          level: critical
                          product: my_app
                          version: 1.0.1
                    - name: monitoring/my_production_app_indicators
                      rules:
                      - alert: average_latency
                        annotations:
                          title: Average Latency
                        expr: average(latency) != 0
                        for: 10m
                        labels:
                          environment: production
                          level: warning
                          product: my_app
                          version: 1.0.1
                `,
			},
		}

		for tn, tc := range testCases {
			t.Run(tn, func(t *testing.T) {
				g := NewGomegaWithT(t)
				p := prometheus.NewConfig()

				expectedAlerts := prometheus_alerts.Document{}
				err := yaml.Unmarshal([]byte(tc.Expected), &expectedAlerts)
				g.Expect(err).NotTo(HaveOccurred())
				for _, i := range tc.Indicators {
					p.Upsert(i)
				}

				alerts := prometheus_alerts.Document{}
				err = yaml.Unmarshal([]byte(p.String()), &alerts)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(alerts.Groups).To(ConsistOf(expectedAlerts.Groups))
			})
		}
	})

	t.Run("it does not render deleted documents", func(t *testing.T) {
		g := NewGomegaWithT(t)
		p := prometheus.NewConfig()

		for _, i := range indicators {
			p.Upsert(i)
		}

		p.Delete(indicators[0])

		g.Expect(p.String()).To(MatchYAML(`
            groups:
            - name: monitoring/my_production_app_indicators
              rules:
              - alert: average_latency
                annotations:
                  title: Average Latency
                expr: average(latency) != 0
                for: 10m
                labels:
                  environment: production
                  level: warning
                  product: my_app
                  version: 1.0.1
        `))
	})

	t.Run("it can handle concurrent reads/writes", func(t *testing.T) {
		p := prometheus.NewConfig()

		for i := 0; i < 100; i++ {
			go p.Upsert(indicators[0])
		}
		for i := 0; i < 100; i++ {
			go p.Delete(indicators[0])
		}
		for i := 0; i < 100; i++ {
			go p.String()
		}
	})
}
