package metricwebhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/wingsofovnia/metrics-webhook/pkg/apis/metrics/v1alpha1"
)

const defaultHttpTimeout = time.Second * 3

type MetricNotificationClient struct {
	httpClient *http.Client
}

func NewDefaultMetricAlertClient() *MetricNotificationClient {
	return NewMetricAlertClient(&http.Client{
		Timeout: defaultHttpTimeout,
	})
}

func NewMetricAlertClient(httpClient *http.Client) *MetricNotificationClient {
	return &MetricNotificationClient{httpClient: httpClient}
}

func (c *MetricNotificationClient) notify(webhookUrl string, report v1alpha1.MetricReport) error {
	reqBodyBytes, err := json.Marshal(report)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", webhookUrl, bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		resBodyStr, _ := ioutil.ReadAll(res.Body)
		return fmt.Errorf("unexpected webhook response: status = %d, body = %s", res.StatusCode, resBodyStr)
	}

	return nil
}
