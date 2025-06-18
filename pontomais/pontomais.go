package pontomais

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Estruturas para parsear o JSON
type Status struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WorkDay struct {
	ID            int64   `json:"id"`
	Date          string  `json:"date"`
	Status        *Status `json:"status"`
	ProcessStatus struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"process_status"`
	AllowExemptionAllowance        bool        `json:"allow_exemption_allowance"`
	LastSolicitationProposalStatus interface{} `json:"last_solicitation_proposal_status"`
}

type ResponseWorkDay struct {
	WorkDays []WorkDay `json:"work_days"`
}

// Estruturas para o ajuste de ponto
type TimeAttribute struct {
	Date   string `json:"date"`
	Time   string `json:"time"`
	Edited bool   `json:"edited"`
}

type Proposal struct {
	Date            string          `json:"date"`
	Motive          string          `json:"motive"`
	TimesAttributes []TimeAttribute `json:"times_attributes"`
	ProposalType    int             `json:"proposal_type"`
}

type Device struct {
	Browser struct {
		Name                string `json:"name"`
		Version             string `json:"version"`
		VersionSearchString string `json:"versionSearchString"`
	} `json:"browser"`
}

type AjustePontoRequest struct {
	Proposal   Proposal `json:"proposal"`
	Path       string   `json:"_path"`
	Device     Device   `json:"_device"`
	AppVersion string   `json:"_appVersion"`
}

// Configuração da API
type PontoMaisConfig struct {
	AccessToken string
	Token       string
	Uid         string
	Client      string
	Uuid        string
}

// Função para obter os dias de trabalho
func GetWorkDays(config PontoMaisConfig, dataInicio, dataFim time.Time) ([]WorkDay, error) {
	// Criando a URL base
	baseURL := "https://api.pontomais.com.br/api/time_card_control/current/work_days"

	// Criando os parâmetros da URL
	params := url.Values{}
	params.Add("start_date", dataInicio.Format("2006-01-02"))
	params.Add("end_date", dataFim.Format("2006-01-02"))
	params.Add("sort_direction", "desc")
	params.Add("sort_property", "date")

	// Construindo a URL completa com os parâmetros
	url_ajuste := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Criando um novo request
	req, err := http.NewRequest("GET", url_ajuste, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar a requisição: %v", err)
	}

	// Adicionando headers
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0")
	req.Header.Add("Access-Token", config.AccessToken)
	req.Header.Add("Token", config.Token)
	req.Header.Add("Api-Version", "2")
	req.Header.Add("Uid", config.Uid)
	req.Header.Add("Client", config.Client)
	req.Header.Add("Uuid", config.Uuid)
	req.Header.Add("sec-ch-ua", `"Microsoft Edge";v="135", "Not-A.Brand";v="8", "Chromium";v="135"`)
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", `"Windows"`)
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-site")

	// Criando um cliente HTTP
	client := &http.Client{}

	// Fazendo a requisição
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer a requisição: %v", err)
	}
	defer resp.Body.Close()

	// Lendo o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o corpo da resposta: %v", err)
	}

	// Parseando o JSON
	var response ResponseWorkDay
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao parsear o JSON: %v", err)
	}

	return response.WorkDays, nil
}

// Função para fazer o ajuste de ponto
func AjustarPonto(config PontoMaisConfig, request AjustePontoRequest) error {
	// URL da API
	url := "https://api.pontomais.com.br/api/time_cards/proposals"

	// Convertendo o request para JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("erro ao converter request para JSON: %v", err)
	}

	// Criando um novo request com o corpo JSON
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("erro ao criar a requisição: %v", err)
	}

	// Adicionando headers
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/135.0.0.0 Safari/537.36 Edg/135.0.0.0")
	req.Header.Add("Access-Token", config.AccessToken)
	req.Header.Add("Token", config.Token)
	req.Header.Add("Api-Version", "2")
	req.Header.Add("Uid", config.Uid)
	req.Header.Add("Client", config.Client)
	req.Header.Add("Origin", "https://app2.pontomais.com.br")
	req.Header.Add("Referer", "https://app2.pontomais.com.br/")
	req.Header.Add("Content-Length", fmt.Sprintf("%d", len(jsonData)))
	req.Header.Add("Uuid", "764c0af2-a116-4075-9d7a-12c3675f840e")
	req.Header.Add("sec-ch-ua", `"Microsoft Edge";v="135", "Not-A.Brand";v="8", "Chromium";v="135"`)
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", `"Windows"`)
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-site")

	// Criando um cliente HTTP
	client := &http.Client{}

	// Fazendo a requisição
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("erro ao fazer a requisição: %v", err)
	}
	defer resp.Body.Close()

	// Lendo o corpo da resposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("erro ao ler o corpo da resposta: %v", err)
	}

	// Verificando o status da resposta
	if resp.StatusCode != http.StatusCreated { // 201 Created
		return fmt.Errorf("erro na requisição: %s - %s", resp.Status, string(body))
	}

	return nil
}

// https://app2.pontomais.com.br/meu-ponto/ajuste/29-04-2025;id=1991631201

// https://api.pontomais.com.br/api/time_cards/proposals
