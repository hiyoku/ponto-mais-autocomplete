package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hiyoku/ponto-mais-autocomplete/pontomais"
)

const (
	defaultAdjustmentMotive      = "Esqueci de apontar os horarios GoAjuste"
	defaultProposalAppVersion    = "0.10.32"
	defaultProposalBrowserName   = "chrome"
	defaultProposalBrowserVer    = "135.0.0.0"
	defaultProposalBrowserSearch = "chrome"
)

var randomMinuteSource = rand.New(rand.NewSource(time.Now().UnixNano()))

type loginFunc func(config pontomais.PontoMaisConfig) (pontomais.LoginResponse, error)
type credentialPromptFunc func(reader io.Reader, writer io.Writer) (string, string, error)

func hasAnyAuthInput(accessToken, uid, client, email, password string) bool {
	return accessToken != "" || uid != "" || client != "" || email != "" || password != ""
}

func hasValidAuthInput(accessToken, uid, client, email, password string) bool {
	return (accessToken != "" && uid != "" && client != "") || (email != "" && password != "")
}

func normalizeTinnovaEmailInput(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return email
	}

	if !strings.Contains(email, "@") {
		return email + "@tinnova.com.br"
	}

	return email
}

func validateTinnovaEmail(email string) error {
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return fmt.Errorf("email inválido")
	}

	if !strings.HasSuffix(strings.ToLower(email), "@tinnova.com.br") {
		return fmt.Errorf("o email deve ser do domínio @tinnova.com.br")
	}

	return nil
}

func readTrimmedLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

	line = strings.TrimSpace(line)
	if err == io.EOF && line == "" {
		return "", io.EOF
	}

	return line, nil
}

func promptForCredentials(reader io.Reader, writer io.Writer) (string, string, error) {
	bufferedReader := bufio.NewReader(reader)

	for {
		fmt.Fprint(writer, "E-mail (@tinnova.com.br): ")
		email, err := readTrimmedLine(bufferedReader)
		if err != nil {
			return "", "", err
		}
		email = normalizeTinnovaEmailInput(email)

		if err := validateTinnovaEmail(email); err != nil {
			fmt.Fprintf(writer, "Erro: %v\n", err)
			continue
		}

		for {
			fmt.Fprint(writer, "Senha: ")
			password, err := readTrimmedLine(bufferedReader)
			if err != nil {
				return "", "", err
			}

			if password == "" {
				fmt.Fprintln(writer, "Erro: a senha não pode ser vazia")
				continue
			}

			return email, password, nil
		}
	}
}

func promptForLastMonths(reader io.Reader, writer io.Writer) (int, error) {
	bufferedReader := bufio.NewReader(reader)

	for {
		fmt.Fprint(writer, "Quantos meses voce deseja corrigir? ")
		value, err := readTrimmedLine(bufferedReader)
		if err != nil {
			return 0, err
		}

		lastMonths, err := strconv.Atoi(value)
		if err != nil || lastMonths <= 0 {
			fmt.Fprintln(writer, "Erro: informe um número inteiro positivo de meses")
			continue
		}

		return lastMonths, nil
	}
}

func waitForUserInteraction(reader io.Reader, writer io.Writer, message string) error {
	bufferedReader := bufio.NewReader(reader)
	if message != "" {
		fmt.Fprintln(writer, message)
	}
	_, err := readTrimmedLine(bufferedReader)
	if err != nil && err != io.EOF {
		return err
	}
	return nil
}

func clearTerminal(writer io.Writer) error {
	_, err := fmt.Fprint(writer, "\033[H\033[2J")
	return err
}

func applyLoginResponse(config pontomais.PontoMaisConfig, loginResponse pontomais.LoginResponse) pontomais.PontoMaisConfig {
	config.AccessToken = loginResponse.Token
	config.Token = loginResponse.Token
	config.Uid = loginResponse.Data.Login
	config.Client = loginResponse.ClientID
	return config
}

func loginWithRetryForPromptedCredentials(config pontomais.PontoMaisConfig, reader io.Reader, writer io.Writer, login loginFunc, prompt credentialPromptFunc) (pontomais.PontoMaisConfig, error) {
	for {
		loginResponse, err := login(config)
		if err == nil {
			return applyLoginResponse(config, loginResponse), nil
		}

		if !errors.Is(err, pontomais.ErrInvalidCredentials) {
			return config, err
		}

		if err := clearTerminal(writer); err != nil {
			return config, err
		}
		fmt.Fprintln(writer, "Credenciais inválidas. Tente novamente.")

		email, password, err := prompt(reader, writer)
		if err != nil {
			return config, err
		}

		config.Email = email
		config.Password = password
	}
}

func wasFlagProvided(args []string, names ...string) bool {
	for _, arg := range args {
		for _, name := range names {
			if arg == name || strings.HasPrefix(arg, name+"=") {
				return true
			}
		}
	}

	return false
}

func validateLastMonths(lastMonths int, args []string) error {
	if !wasFlagProvided(args, "--last-months", "-lm") {
		return nil
	}

	if lastMonths <= 0 {
		return fmt.Errorf("o parâmetro --last-months/-lm deve ser um inteiro positivo")
	}

	return nil
}

func shouldScanMonth(processedMonths, lastMonths int) bool {
	return lastMonths <= 0 || processedMonths < lastMonths
}

func monthRange(year int, month time.Month, loc *time.Location) (time.Time, time.Time) {
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	lastDay := firstDay.AddDate(0, 1, -1)
	return firstDay, lastDay
}

func previousMonth(year int, month time.Month) (int, time.Month) {
	month--
	if month == 0 {
		return year - 1, time.December
	}

	return year, month
}

func collectAdjustmentCandidates(workDays []pontomais.WorkDay) (int, int, []pontomais.WorkDay) {
	faltas := 0
	paraAjuste := 0
	candidates := make([]pontomais.WorkDay, 0)

	for _, workDay := range workDays {
		if workDay.Status == nil || workDay.Status.Name != "Falta" {
			continue
		}

		faltas++
		if workDay.LastSolicitationProposalStatus != nil {
			continue
		}

		paraAjuste++
		candidates = append(candidates, workDay)
	}

	return faltas, paraAjuste, candidates
}

func randomMinute() int {
	return randomMinuteSource.Intn(16)
}

func randomInRange(min, max int) int {
	return min + randomMinuteSource.Intn(max-min+1)
}

func formatHourMinute(hour, minute int) string {
	return fmt.Sprintf("%02d:%02d", hour, minute)
}

func formatClock(t time.Time) string {
	return t.Format("15:04")
}

func buildHumanLikeTimes(randomMinute func() int, randomLunchDuration func(min, max int) int) []string {
	base := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	startMinute := randomMinute()
	lunchOutMinute := randomMinute()
	lunchDurationMinutes := randomLunchDuration(60, 75)

	start := base.Add(time.Hour*8 + time.Duration(startMinute)*time.Minute)
	lunchOut := base.Add(time.Hour*12 + time.Duration(lunchOutMinute)*time.Minute)
	lunchIn := lunchOut.Add(time.Duration(lunchDurationMinutes) * time.Minute)
	end := start.Add(8*time.Hour + time.Duration(lunchDurationMinutes)*time.Minute)

	return []string{
		formatClock(start),
		formatClock(lunchOut),
		formatClock(lunchIn),
		formatClock(end),
	}
}

func buildAjusteRequest(workDay pontomais.WorkDay) pontomais.AjustePontoRequest {
	appointmentTimes := buildHumanLikeTimes(randomMinute, randomInRange)
	times := make([]pontomais.TimeAttribute, 0, len(appointmentTimes))
	for _, t := range appointmentTimes {
		times = append(times, pontomais.TimeAttribute{
			Date:   workDay.Date,
			Time:   t,
			Edited: true,
		})
	}

	return pontomais.AjustePontoRequest{
		Proposal: pontomais.Proposal{
			Date:            workDay.Date,
			Motive:          defaultAdjustmentMotive,
			TimesAttributes: times,
			ProposalType:    1,
		},
		Path: fmt.Sprintf("/meu-ponto/ajuste/%s;id=%d", workDay.Date, workDay.ID),
		Device: pontomais.Device{
			Browser: struct {
				Name                string `json:"name"`
				Version             string `json:"version"`
				VersionSearchString string `json:"versionSearchString"`
			}{
				Name:                defaultProposalBrowserName,
				Version:             defaultProposalBrowserVer,
				VersionSearchString: defaultProposalBrowserSearch,
			},
		},
		AppVersion: defaultProposalAppVersion,
	}
}

func main() {
	// Definindo os parâmetros
	accessToken := flag.String("access-token", "", "Token de acesso do PontoMais")
	uid := flag.String("uid", "", "UID do usuário")
	client := flag.String("client", "", "ID do cliente")
	email := flag.String("email", "", "E-mail do usuário")
	password := flag.String("password", "", "Senha do usuário")
	lastMonths := flag.Int("last-months", 0, "Quantidade de meses para buscar, incluindo o mês atual")
	flag.IntVar(lastMonths, "lm", 0, "Quantidade de meses para buscar, incluindo o mês atual")
	flag.Parse()

	promptedCredentials := false

	if err := validateLastMonths(*lastMonths, os.Args[1:]); err != nil {
		fmt.Println("Erro:", err)
		fmt.Println("Exemplo: go run main.go --email=SEU_EMAIL --password=SEU_PASSWORD --last-months=3")
		return
	}

	if !hasAnyAuthInput(*accessToken, *uid, *client, *email, *password) {
		promptedEmail, promptedPassword, err := promptForCredentials(os.Stdin, os.Stdout)
		if err != nil {
			fmt.Printf("Erro ao ler credenciais: %v\n", err)
			return
		}

		*email = promptedEmail
		*password = promptedPassword
		promptedCredentials = true
	}

	// Verificando se os parâmetros obrigatórios foram fornecidos
	if !hasValidAuthInput(*accessToken, *uid, *client, *email, *password) {
		fmt.Println("Uso: go run main.go --access-token=SEU_ACCESS_TOKEN --uid=SEU_UID --client=SEU_CLIENT")
		fmt.Println("   ou: go run main.go --email=SEU_EMAIL --password=SEU_PASSWORD")
		fmt.Println("Parâmetros obrigatórios:")
		fmt.Println("  --access-token, --uid, --client  (ou)")
		fmt.Println("  --email, --password")
		return
	}

	// Configuração da API
	config := pontomais.PontoMaisConfig{
		AccessToken: *accessToken,
		Token:       *accessToken, // Usando o mesmo access token como token
		Uid:         *uid,
		Client:      *client,
		Uuid:        "",
		Email:       *email,
		Password:    *password,
	}

	if config.AccessToken == "" {
		if promptedCredentials {
			updatedConfig, err := loginWithRetryForPromptedCredentials(config, os.Stdin, os.Stdout, pontomais.GetAccessToken, promptForCredentials)
			if err != nil {
				fmt.Printf("Erro ao obter token de acesso: %v\n", err)
				return
			}
			config = updatedConfig

			if !wasFlagProvided(os.Args[1:], "--last-months", "-lm") {
				promptedLastMonths, err := promptForLastMonths(os.Stdin, os.Stdout)
				if err != nil {
					fmt.Printf("Erro ao ler quantidade de meses: %v\n", err)
					return
				}
				*lastMonths = promptedLastMonths
			}
		} else {
			loginResponse, err := pontomais.GetAccessToken(config)
			if err != nil {
				fmt.Printf("Erro ao obter token de acesso: %v\n", err)
				return
			}
			config = applyLoginResponse(config, loginResponse)
		}
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
	processedMonths := 0

	// Loop para percorrer os meses anteriores
	for shouldScanMonth(processedMonths, *lastMonths) {
		// Obtendo as datas do mês atual
		primeiroDiaMes, ultimoDiaMes := monthRange(anoAtual, mesAtual, time.Local)

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
		faltasNoMes, paraAjusteNoMes, candidates := collectAdjustmentCandidates(workDays)
		todosWorkDays = append(todosWorkDays, candidates...)

		totalFaltas += faltasNoMes
		totalParaAjuste += paraAjusteNoMes

		fmt.Printf("Mês %d/%d: %d dias, %d faltas, %d para ajuste\n",
			mesAtual, anoAtual, len(workDays), faltasNoMes, paraAjusteNoMes)
		processedMonths++

		// Indo para o mês anterior
		anoAtual, mesAtual = previousMonth(anoAtual, mesAtual)
	}

	// Exibindo o resumo
	fmt.Printf("\nResumo:\n")
	fmt.Printf("Total de dias analisados: %d\n", totalDias)
	fmt.Printf("Total de faltas encontradas: %d\n", totalFaltas)
	fmt.Printf("Total de faltas para ajuste: %d\n", totalParaAjuste)

	// Ajustando cada dia de falta
	for _, workDay := range todosWorkDays {
		// Criando o ajuste para o dia
		ajuste := buildAjusteRequest(workDay)

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
	if err := waitForUserInteraction(os.Stdin, os.Stdout, "Pressione Enter para continuar..."); err != nil {
		fmt.Printf("Erro ao aguardar interação do usuário: %v\n", err)
		return
	}
	fmt.Println("Obrigado por usar a gambiarra do Hideki")
	if err := waitForUserInteraction(os.Stdin, os.Stdout, "Pressione Enter para encerrar..."); err != nil {
		fmt.Printf("Erro ao aguardar interação do usuário: %v\n", err)
		return
	}
}
