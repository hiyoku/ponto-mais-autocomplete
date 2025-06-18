package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/hiyoku/ponto-mais-autocomplete/pontomais"
)

func main() {
	// Definindo os parâmetros
	accessToken := flag.String("access-token", "", "Access Token do PontoMais")
	uid := flag.String("uid", "", "UID do usuário")
	client := flag.String("client", "", "Client ID")
	flag.Parse()

	// Verificando se os parâmetros obrigatórios foram fornecidos
	if *accessToken == "" || *uid == "" || *client == "" {
		fmt.Println("Uso: go run main.go --access-token=SEU_ACCESS_TOKEN --uid=SEU_UID --client=SEU_CLIENT")
		fmt.Println("Parâmetros obrigatórios:")
		fmt.Println("  --access-token: Access Token do PontoMais")
		fmt.Println("  --uid: UID do usuário")
		fmt.Println("  --client: Client ID")
		return
	}

	// Configuração da API
	config := pontomais.PontoMaisConfig{
		AccessToken: *accessToken,
		Token:       *accessToken, // Usando o mesmo access token como token
		Uid:         *uid,
		Client:      *client,
		Uuid:        "",
	}
	// Lista para armazenar todos os WorkDays
	var todosWorkDays []pontomais.WorkDay

	// Começando do mês atual
	dataAtual := time.Now()
	mesAtual := dataAtual.Month()
	anoAtual := dataAtual.Year()

	// Contadores
	totalDias := 0
	totalFaltas := 0
	totalParaAjuste := 0

	// Loop para percorrer os meses anteriores
	for {
		// Obtendo as datas do mês atual
		primeiroDiaMes := time.Date(anoAtual, mesAtual, 1, 0, 0, 0, 0, time.Local)
		ultimoDiaMes := primeiroDiaMes.AddDate(0, 1, -1)

		// Obtendo os dias de trabalho do mês
		workDays, err := pontomais.GetWorkDays(config, primeiroDiaMes, ultimoDiaMes)
		if err != nil {
			fmt.Printf("Erro ao obter dados do mês %d/%d: %v\n", mesAtual, anoAtual, err)
			break
		}

		// Se não houver dados, paramos o loop
		if len(workDays) == 0 {
			fmt.Printf("Nenhum dado encontrado para o mês %d/%d\n", mesAtual, anoAtual)
			break
		}

		// Atualizando contadores
		totalDias += len(workDays)
		faltasNoMes := 0
		paraAjusteNoMes := 0

		// Filtrando apenas os registros com status "Falta" e last_solicitation_proposal_status null
		for _, workDay := range workDays {
			if workDay.Status != nil && workDay.Status.Name == "Falta" {
				faltasNoMes++
				if workDay.LastSolicitationProposalStatus == nil {
					paraAjusteNoMes++
					todosWorkDays = append(todosWorkDays, workDay)
				}
			}
		}

		totalFaltas += faltasNoMes
		totalParaAjuste += paraAjusteNoMes

		fmt.Printf("Mês %d/%d: %d dias, %d faltas, %d para ajuste\n",
			mesAtual, anoAtual, len(workDays), faltasNoMes, paraAjusteNoMes)

		// Indo para o mês anterior
		mesAtual--
		if mesAtual == 0 {
			mesAtual = 12
			anoAtual--
		}
	}

	// Exibindo o resumo
	fmt.Printf("\nResumo:\n")
	fmt.Printf("Total de dias analisados: %d\n", totalDias)
	fmt.Printf("Total de faltas encontradas: %d\n", totalFaltas)
	fmt.Printf("Total de faltas para ajuste: %d\n", totalParaAjuste)

	// Ajustando cada dia de falta
	for _, workDay := range todosWorkDays {
		// Criando o ajuste para o dia
		ajuste := pontomais.AjustePontoRequest{
			Proposal: pontomais.Proposal{
				Date:   workDay.Date,
				Motive: "Esqueci de apontar os horarios GoAjuste",
				TimesAttributes: []pontomais.TimeAttribute{
					{Date: workDay.Date, Time: "08:00", Edited: true},
					{Date: workDay.Date, Time: "12:00", Edited: true},
					{Date: workDay.Date, Time: "13:00", Edited: true},
					{Date: workDay.Date, Time: "17:00", Edited: true},
				},
				ProposalType: 1,
			},
			Path: fmt.Sprintf("/meu-ponto/ajuste/%s;id=%d", workDay.Date, workDay.ID),
			Device: pontomais.Device{
				Browser: struct {
					Name                string `json:"name"`
					Version             string `json:"version"`
					VersionSearchString string `json:"versionSearchString"`
				}{
					Name:                "chrome",
					Version:             "135.0.0.0",
					VersionSearchString: "chrome",
				},
			},
			AppVersion: "0.10.32",
		}

		// Fazendo o ajuste de ponto
		fmt.Printf("Ajustando ponto para o dia %s...\n", workDay.Date)
		err := pontomais.AjustarPonto(config, ajuste)
		if err != nil {
			fmt.Printf("Erro ao ajustar ponto do dia %s: %v\n", workDay.Date, err)
			continue
		}
		fmt.Printf("Ponto ajustado com sucesso para o dia %s\n", workDay.Date)
	}

	fmt.Println("\nProcesso de ajuste de pontos concluído!")
}
