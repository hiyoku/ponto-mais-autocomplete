package main

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hiyoku/ponto-mais-autocomplete/pontomais"
)

func TestHasAnyAuthInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		accessToken string
		uid         string
		client      string
		email       string
		password    string
		want        bool
	}{
		{
			name: "all empty",
			want: false,
		},
		{
			name:  "email only counts as some input",
			email: "user@tinnova.com.br",
			want:  true,
		},
		{
			name:        "token only counts as some input",
			accessToken: "token",
			want:        true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := hasAnyAuthInput(tc.accessToken, tc.uid, tc.client, tc.email, tc.password)
			if got != tc.want {
				t.Fatalf("hasAnyAuthInput() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHasValidAuthInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		accessToken string
		uid         string
		client      string
		email       string
		password    string
		want        bool
	}{
		{
			name:        "valid token auth",
			accessToken: "token",
			uid:         "uid",
			client:      "client",
			want:        true,
		},
		{
			name:     "valid email auth",
			email:    "user@example.com",
			password: "secret",
			want:     true,
		},
		{
			name:        "missing token field",
			accessToken: "token",
			uid:         "uid",
			want:        false,
		},
		{
			name:     "missing password",
			email:    "user@example.com",
			password: "",
			want:     false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := hasValidAuthInput(tc.accessToken, tc.uid, tc.client, tc.email, tc.password)
			if got != tc.want {
				t.Fatalf("hasValidAuthInput() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateTinnovaEmail(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid tinnova email",
			email:   "user@tinnova.com.br",
			wantErr: false,
		},
		{
			name:    "uppercase domain still accepted",
			email:   "user@TINNOVA.COM.BR",
			wantErr: false,
		},
		{
			name:    "invalid format",
			email:   "not-an-email",
			wantErr: true,
		},
		{
			name:    "wrong domain",
			email:   "user@gmail.com",
			wantErr: true,
		},
		{
			name:    "name plus address should fail strict check",
			email:   "User <user@tinnova.com.br>",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateTinnovaEmail(tc.email)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateTinnovaEmail() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestNormalizeTinnovaEmailInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "username only appends domain",
			input: "user",
			want:  "user@tinnova.com.br",
		},
		{
			name:  "full email is preserved",
			input: "user@tinnova.com.br",
			want:  "user@tinnova.com.br",
		},
		{
			name:  "input is trimmed",
			input: "  user  ",
			want:  "user@tinnova.com.br",
		},
		{
			name:  "other domain is preserved for later validation",
			input: "user@gmail.com",
			want:  "user@gmail.com",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeTinnovaEmailInput(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeTinnovaEmailInput() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadTrimmedLine(t *testing.T) {
	t.Parallel()

	got, err := readTrimmedLine(bufio.NewReader(strings.NewReader("  value  \n")))
	if err != nil {
		t.Fatalf("readTrimmedLine() error = %v", err)
	}

	if got != "value" {
		t.Fatalf("readTrimmedLine() = %q, want %q", got, "value")
	}
}

func TestPromptForCredentials(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("user@@tinnova.com.br\nuser@gmail.com\nuser\n\ncorrect-password\n")
	output := &bytes.Buffer{}

	email, password, err := promptForCredentials(input, output)
	if err != nil {
		t.Fatalf("promptForCredentials() error = %v", err)
	}

	if email != "user@tinnova.com.br" {
		t.Fatalf("email = %q, want %q", email, "user@tinnova.com.br")
	}

	if password != "correct-password" {
		t.Fatalf("password = %q, want %q", password, "correct-password")
	}

	rendered := output.String()
	expectedSnippets := []string{
		"E-mail (@tinnova.com.br): ",
		"Erro: email inválido",
		"Erro: o email deve ser do domínio @tinnova.com.br",
		"Senha: ",
		"Erro: a senha não pode ser vazia",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("prompt output %q does not contain %q", rendered, snippet)
		}
	}
}

func TestPromptForCredentialsEOF(t *testing.T) {
	t.Parallel()

	_, _, err := promptForCredentials(strings.NewReader(""), io.Discard)
	if err == nil {
		t.Fatalf("expected EOF error, got nil")
	}
}

func TestPromptForLastMonths(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("zero\n0\n-1\n3\n")
	output := &bytes.Buffer{}

	lastMonths, err := promptForLastMonths(input, output)
	if err != nil {
		t.Fatalf("promptForLastMonths() error = %v", err)
	}

	if lastMonths != 3 {
		t.Fatalf("lastMonths = %d, want 3", lastMonths)
	}

	rendered := output.String()
	expectedSnippets := []string{
		"Quantos meses voce deseja corrigir? ",
		"Erro: informe um número inteiro positivo de meses",
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(rendered, snippet) {
			t.Fatalf("prompt output %q does not contain %q", rendered, snippet)
		}
	}
}

func TestPromptForLastMonthsEOF(t *testing.T) {
	t.Parallel()

	_, err := promptForLastMonths(strings.NewReader(""), io.Discard)
	if err == nil {
		t.Fatalf("expected EOF error, got nil")
	}
}

func TestWaitForUserInteraction(t *testing.T) {
	t.Parallel()

	output := &bytes.Buffer{}
	err := waitForUserInteraction(strings.NewReader("\n"), output, "Pressione Enter para continuar...")
	if err != nil {
		t.Fatalf("waitForUserInteraction() error = %v", err)
	}

	if output.String() != "Pressione Enter para continuar...\n" {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestWaitForUserInteractionEOFWithoutMessage(t *testing.T) {
	t.Parallel()

	err := waitForUserInteraction(strings.NewReader(""), io.Discard, "")
	if err != nil {
		t.Fatalf("waitForUserInteraction() error = %v", err)
	}
}

func TestClearTerminal(t *testing.T) {
	t.Parallel()

	output := &bytes.Buffer{}
	if err := clearTerminal(output); err != nil {
		t.Fatalf("clearTerminal() error = %v", err)
	}

	if output.String() != "\033[H\033[2J" {
		t.Fatalf("clearTerminal() wrote %q", output.String())
	}
}

func TestApplyLoginResponse(t *testing.T) {
	t.Parallel()

	config := pontomais.PontoMaisConfig{Email: "user@tinnova.com.br", Password: "secret"}
	response := pontomais.LoginResponse{
		Token:    "token-1",
		ClientID: "client-1",
	}
	response.Data.Login = "uid-1"

	got := applyLoginResponse(config, response)
	if got.AccessToken != "token-1" || got.Token != "token-1" || got.Client != "client-1" || got.Uid != "uid-1" {
		t.Fatalf("applyLoginResponse() = %+v", got)
	}
}

func TestLoginWithRetryForPromptedCredentials(t *testing.T) {
	t.Parallel()

	config := pontomais.PontoMaisConfig{
		Email:    "wrong@tinnova.com.br",
		Password: "wrong",
	}
	output := &bytes.Buffer{}
	loginCalls := 0
	promptCalls := 0

	got, err := loginWithRetryForPromptedCredentials(
		config,
		strings.NewReader(""),
		output,
		func(config pontomais.PontoMaisConfig) (pontomais.LoginResponse, error) {
			loginCalls++
			if loginCalls == 1 {
				return pontomais.LoginResponse{}, pontomais.ErrInvalidCredentials
			}

			response := pontomais.LoginResponse{
				Token:    "token-2",
				ClientID: "client-2",
			}
			response.Data.Login = "uid-2"
			if config.Email != "retry@tinnova.com.br" || config.Password != "retry-password" {
				t.Fatalf("unexpected retried config: %+v", config)
			}
			return response, nil
		},
		func(reader io.Reader, writer io.Writer) (string, string, error) {
			promptCalls++
			return "retry@tinnova.com.br", "retry-password", nil
		},
	)
	if err != nil {
		t.Fatalf("loginWithRetryForPromptedCredentials() error = %v", err)
	}

	if loginCalls != 2 {
		t.Fatalf("login calls = %d, want 2", loginCalls)
	}

	if promptCalls != 1 {
		t.Fatalf("prompt calls = %d, want 1", promptCalls)
	}

	if got.AccessToken != "token-2" || got.Token != "token-2" || got.Client != "client-2" || got.Uid != "uid-2" {
		t.Fatalf("unexpected updated config: %+v", got)
	}

	rendered := output.String()
	if !strings.Contains(rendered, "\033[H\033[2J") || !strings.Contains(rendered, "Credenciais inválidas. Tente novamente.") {
		t.Fatalf("retry output = %q", rendered)
	}
}

func TestLoginWithRetryForPromptedCredentialsStopsOnNonAuthError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("erro de rede")
	_, err := loginWithRetryForPromptedCredentials(
		pontomais.PontoMaisConfig{},
		strings.NewReader(""),
		io.Discard,
		func(config pontomais.PontoMaisConfig) (pontomais.LoginResponse, error) {
			return pontomais.LoginResponse{}, expectedErr
		},
		func(reader io.Reader, writer io.Writer) (string, string, error) {
			t.Fatalf("prompt should not be called on non-auth error")
			return "", "", nil
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestWasFlagProvided(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "long flag with separate value",
			args: []string{"--last-months", "3"},
			want: true,
		},
		{
			name: "short flag with assignment",
			args: []string{"-lm=3"},
			want: true,
		},
		{
			name: "flag absent",
			args: []string{"--email=user@example.com"},
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := wasFlagProvided(tc.args, "--last-months", "-lm")
			if got != tc.want {
				t.Fatalf("wasFlagProvided() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateLastMonths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		lastMonths int
		args       []string
		wantErr    bool
	}{
		{
			name:       "flag not provided keeps unlimited mode",
			lastMonths: 0,
			args:       []string{"--email=user@example.com"},
			wantErr:    false,
		},
		{
			name:       "valid long flag",
			lastMonths: 3,
			args:       []string{"--last-months=3"},
			wantErr:    false,
		},
		{
			name:       "valid short flag",
			lastMonths: 2,
			args:       []string{"-lm", "2"},
			wantErr:    false,
		},
		{
			name:       "zero is invalid when provided",
			lastMonths: 0,
			args:       []string{"--last-months=0"},
			wantErr:    true,
		},
		{
			name:       "negative is invalid when provided",
			lastMonths: -1,
			args:       []string{"-lm=-1"},
			wantErr:    true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validateLastMonths(tc.lastMonths, tc.args)
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateLastMonths() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestMonthRange(t *testing.T) {
	t.Parallel()

	loc := time.FixedZone("UTC-3", -3*60*60)
	firstDay, lastDay := monthRange(2026, time.February, loc)

	if firstDay.Format(time.RFC3339) != "2026-02-01T00:00:00-03:00" {
		t.Fatalf("unexpected first day: %s", firstDay.Format(time.RFC3339))
	}

	if lastDay.Format(time.RFC3339) != "2026-02-28T00:00:00-03:00" {
		t.Fatalf("unexpected last day: %s", lastDay.Format(time.RFC3339))
	}
}

func TestPreviousMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		year      int
		month     time.Month
		wantYear  int
		wantMonth time.Month
	}{
		{
			name:      "middle of year",
			year:      2026,
			month:     time.July,
			wantYear:  2026,
			wantMonth: time.June,
		},
		{
			name:      "year wrap",
			year:      2026,
			month:     time.January,
			wantYear:  2025,
			wantMonth: time.December,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotYear, gotMonth := previousMonth(tc.year, tc.month)
			if gotYear != tc.wantYear || gotMonth != tc.wantMonth {
				t.Fatalf("previousMonth() = (%d, %s), want (%d, %s)", gotYear, gotMonth, tc.wantYear, tc.wantMonth)
			}
		})
	}
}

func TestShouldScanMonth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		processedMonths int
		lastMonths      int
		want            bool
	}{
		{
			name:            "unlimited when flag omitted",
			processedMonths: 12,
			lastMonths:      0,
			want:            true,
		},
		{
			name:            "first month within limit",
			processedMonths: 0,
			lastMonths:      3,
			want:            true,
		},
		{
			name:            "last allowed month still runs",
			processedMonths: 2,
			lastMonths:      3,
			want:            true,
		},
		{
			name:            "stops after limit",
			processedMonths: 3,
			lastMonths:      3,
			want:            false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := shouldScanMonth(tc.processedMonths, tc.lastMonths)
			if got != tc.want {
				t.Fatalf("shouldScanMonth() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCollectAdjustmentCandidates(t *testing.T) {
	t.Parallel()

	workDays := []pontomais.WorkDay{
		{
			ID:     1,
			Date:   "2026-03-10",
			Status: &pontomais.Status{Name: "Falta"},
		},
		{
			ID:                             2,
			Date:                           "2026-03-11",
			Status:                         &pontomais.Status{Name: "Falta"},
			LastSolicitationProposalStatus: "pending",
		},
		{
			ID:     3,
			Date:   "2026-03-12",
			Status: &pontomais.Status{Name: "Presente"},
		},
		{
			ID:   4,
			Date: "2026-03-13",
		},
	}

	faltas, paraAjuste, candidates := collectAdjustmentCandidates(workDays)

	if faltas != 2 {
		t.Fatalf("faltas = %d, want 2", faltas)
	}

	if paraAjuste != 1 {
		t.Fatalf("paraAjuste = %d, want 1", paraAjuste)
	}

	if len(candidates) != 1 || candidates[0].ID != 1 {
		t.Fatalf("candidates = %+v, want only workday ID 1", candidates)
	}
}

func TestFormatHourMinute(t *testing.T) {
	t.Parallel()

	if got := formatHourMinute(8, 5); got != "08:05" {
		t.Fatalf("formatHourMinute() = %s, want 08:05", got)
	}
}

func TestFormatClock(t *testing.T) {
	t.Parallel()

	base := time.Date(2000, time.January, 1, 16, 42, 0, 0, time.UTC)
	if got := formatClock(base); got != "16:42" {
		t.Fatalf("formatClock() = %s, want 16:42", got)
	}
}

func TestBuildHumanLikeTimes(t *testing.T) {
	t.Parallel()

	minutes := []int{7, 11}
	minuteIndex := 0

	got := buildHumanLikeTimes(func() int {
		value := minutes[minuteIndex]
		minuteIndex++
		return value
	}, func(min, max int) int {
		if min != 60 || max != 75 {
			t.Fatalf("unexpected lunch duration range: %d-%d", min, max)
		}
		return 64
	})

	want := []string{"08:07", "12:11", "13:15", "17:11"}
	if len(got) != len(want) {
		t.Fatalf("times length = %d, want %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("time %d = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestBuildHumanLikeTimesGuaranteesEightHours(t *testing.T) {
	t.Parallel()

	for startMinute := 0; startMinute <= 15; startMinute++ {
		for lunchOutMinute := 0; lunchOutMinute <= 15; lunchOutMinute++ {
			for lunchDuration := 60; lunchDuration <= 75; lunchDuration++ {
				minutes := []int{startMinute, lunchOutMinute}
				minuteIndex := 0

				times := buildHumanLikeTimes(func() int {
					value := minutes[minuteIndex]
					minuteIndex++
					return value
				}, func(min, max int) int {
					if min != 60 || max != 75 {
						t.Fatalf("unexpected lunch duration range: %d-%d", min, max)
					}
					return lunchDuration
				})

				start := mustClockTime(t, times[0])
				lunchOut := mustClockTime(t, times[1])
				lunchIn := mustClockTime(t, times[2])
				end := mustClockTime(t, times[3])

				worked := lunchOut.Sub(start) + end.Sub(lunchIn)
				if worked != 8*time.Hour {
					t.Fatalf("worked duration = %s for times %v, want 8h", worked, times)
				}
			}
		}
	}
}

func TestBuildHumanLikeTimesHumanRanges(t *testing.T) {
	t.Parallel()

	for startMinute := 0; startMinute <= 15; startMinute++ {
		for lunchOutMinute := 0; lunchOutMinute <= 15; lunchOutMinute++ {
			for lunchDuration := 60; lunchDuration <= 75; lunchDuration++ {
				minutes := []int{startMinute, lunchOutMinute}
				minuteIndex := 0

				times := buildHumanLikeTimes(func() int {
					value := minutes[minuteIndex]
					minuteIndex++
					return value
				}, func(min, max int) int {
					return lunchDuration
				})

				start := mustClockTime(t, times[0])
				lunchOut := mustClockTime(t, times[1])
				lunchIn := mustClockTime(t, times[2])
				end := mustClockTime(t, times[3])

				if start.Hour() != 8 {
					t.Fatalf("start hour = %d, want 8 for %v", start.Hour(), times)
				}

				if lunchOut.Hour() != 12 {
					t.Fatalf("lunch out hour = %d, want 12 for %v", lunchOut.Hour(), times)
				}

				lunchWindow := lunchIn.Sub(lunchOut)
				if lunchWindow < time.Hour || lunchWindow > time.Hour+15*time.Minute {
					t.Fatalf("lunch window = %s, want between 1h and 1h15 for %v", lunchWindow, times)
				}

				if end.Sub(start) != 8*time.Hour+lunchWindow {
					t.Fatalf("end time mismatch for %v", times)
				}

				if start.Minute() < 0 || start.Minute() > 15 {
					t.Fatalf("start minute out of range for %v", times)
				}

				if lunchOut.Minute() < 0 || lunchOut.Minute() > 15 {
					t.Fatalf("lunch out minute out of range for %v", times)
				}

				if end.Hour() < 17 || end.Hour() > 17 {
					t.Fatalf("end hour = %d, want 17 for %v", end.Hour(), times)
				}
			}
		}
	}
}

func mustClockTime(t *testing.T, clock string) time.Time {
	t.Helper()

	parsed, err := time.Parse("15:04", clock)
	if err != nil {
		t.Fatalf("parse time %q: %v", clock, err)
	}

	return parsed
}

func TestBuildAjusteRequest(t *testing.T) {
	t.Parallel()

	workDay := pontomais.WorkDay{
		ID:   42,
		Date: "2026-03-19",
	}

	oldRandomizer := randomMinuteSource
	randomMinuteSource = rand.New(rand.NewSource(1))
	defer func() {
		randomMinuteSource = oldRandomizer
	}()

	got := buildAjusteRequest(workDay)

	if got.Proposal.Date != workDay.Date {
		t.Fatalf("proposal date = %s, want %s", got.Proposal.Date, workDay.Date)
	}

	if got.Proposal.Motive != defaultAdjustmentMotive {
		t.Fatalf("proposal motive = %s, want %s", got.Proposal.Motive, defaultAdjustmentMotive)
	}

	if got.Proposal.ProposalType != 1 {
		t.Fatalf("proposal type = %d, want 1", got.Proposal.ProposalType)
	}

	if got.Path != "/meu-ponto/ajuste/2026-03-19;id=42" {
		t.Fatalf("path = %s", got.Path)
	}

	if got.AppVersion != defaultProposalAppVersion {
		t.Fatalf("app version = %s, want %s", got.AppVersion, defaultProposalAppVersion)
	}

	if got.Device.Browser.Name != defaultProposalBrowserName {
		t.Fatalf("browser name = %s, want %s", got.Device.Browser.Name, defaultProposalBrowserName)
	}

	if len(got.Proposal.TimesAttributes) != 4 {
		t.Fatalf("times length = %d, want 4", len(got.Proposal.TimesAttributes))
	}

	gotTimes := make([]string, 0, len(got.Proposal.TimesAttributes))
	for i, timeAttr := range got.Proposal.TimesAttributes {
		if timeAttr.Date != workDay.Date {
			t.Fatalf("time attr %d date = %s, want %s", i, timeAttr.Date, workDay.Date)
		}
		if !timeAttr.Edited {
			t.Fatalf("time attr %d edited = false, want true", i)
		}
		gotTimes = append(gotTimes, timeAttr.Time)
	}

	if gotTimes[0][:2] != "08" || gotTimes[1][:2] != "12" || gotTimes[3][:2] != "17" {
		t.Fatalf("unexpected hour layout: %v", gotTimes)
	}

	start := mustClockTime(t, gotTimes[0])
	lunchOut := mustClockTime(t, gotTimes[1])
	lunchIn := mustClockTime(t, gotTimes[2])
	end := mustClockTime(t, gotTimes[3])

	worked := lunchOut.Sub(start) + end.Sub(lunchIn)
	if worked != 8*time.Hour {
		t.Fatalf("worked duration = %s for times %v, want 8h", worked, gotTimes)
	}

	lunchWindow := lunchIn.Sub(lunchOut)
	if lunchWindow < time.Hour || lunchWindow > time.Hour+15*time.Minute {
		t.Fatalf("lunch window = %s, want between 1h and 1h15 for %v", lunchWindow, gotTimes)
	}

	if end.Sub(start) != 8*time.Hour+lunchWindow {
		t.Fatalf("exit time does not reflect 8h worked for %v", gotTimes)
	}
}
