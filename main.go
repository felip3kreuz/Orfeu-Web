package main

import (
	"bufio"
	crand "crypto/rand"
	"embed"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const version = "2.0.0-rc1.8"
const back = "__VOLTAR__"

var forceClassic = "false"
var diagConsole = "false"
var serverMode = "false"
var storageRoot string

//go:embed catalogo_negocios.json catalogo_insumos.json
var embeddedFS embed.FS

var input = bufio.NewReader(os.Stdin)
var rng = rand.New(rand.NewSource(time.Now().UnixNano()))
var helpTopic = "geral"
var colorCapable = false
var colorEnabled = false

const uiWidth = 76

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiDim     = "\033[2m"
	ansiCyan    = "\033[36m"
	ansiBlue    = "\033[94m"
	ansiGreen   = "\033[92m"
	ansiYellow  = "\033[93m"
	ansiRed     = "\033[91m"
	ansiMagenta = "\033[95m"
	ansiWhite   = "\033[97m"
)

// ---------- Dados ----------

// Canvas Persona adotado pelo material do JED.

func digitalModelLabel(e *Empresa) string {
	switch e.ModeloDigital {
	case "afiliacao":
		return "COMISSÃO MÉDIA"
	case "plr":
		return "RECEITA MÉDIA"
	case "dropshipping":
		return "PREÇO MÉDIO"
	case "influencia":
		return "RECEITA / CAMPANHA"
	default:
		return "PREÇO MÉDIO"
	}
}

var cenariosBase = []Cenario{
	{Nome: "Mercado estável", Alcance: 1.00, Conversao: 1.00, Oscilacao: 0.08, EventoNegativoExtra: 0.00},
	{Nome: "Mercado aquecido", Alcance: 1.10, Conversao: 1.04, Oscilacao: 0.10, EventoNegativoExtra: -0.05},
	{Nome: "Desaceleração econômica", Alcance: 0.90, Conversao: 0.92, Oscilacao: 0.12, EventoNegativoExtra: 0.08},
	{Nome: "Concorrência intensa", Alcance: 0.96, Conversao: 0.94, Oscilacao: 0.10, EventoNegativoExtra: 0.05},
}

// ---------- Sistema / arquivos ----------

func appDir() string {
	exe, err := os.Executable()
	if err == nil {
		return filepath.Dir(exe)
	}
	wd, _ := os.Getwd()
	return wd
}

func storageBase() string {
	if storageRoot != "" {
		return storageRoot
	}
	return appDir()
}

func dataDir() string      { return filepath.Join(storageBase(), "dados") }
func reportsDir() string   { return filepath.Join(storageBase(), "relatorios") }
func scenariosDir() string { return filepath.Join(dataDir(), "cenarios") }
func classesDir() string   { return filepath.Join(dataDir(), "turmas") }
func backupsDir() string   { return filepath.Join(dataDir(), "backups") }

func ensureDirsAt(root string) error {
	storageRoot = root
	dirs := []string{dataDir(), reportsDir(), scenariosDir(), classesDir(), backupsDir(), filepath.Join(dataDir(), "empresas")}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

func ensureDirs() error {
	// Prefere manter os dados ao lado do executável, como nas versões portáteis.
	if err := ensureDirsAt(appDir()); err == nil {
		return nil
	}
	// Em pastas protegidas do Windows, usa LOCALAPPDATA automaticamente.
	if runtime.GOOS == "windows" {
		root := os.Getenv("LOCALAPPDATA")
		if root == "" {
			root = os.TempDir()
		}
		return ensureDirsAt(filepath.Join(root, "JED Simulador"))
	}
	return ensureDirsAt(filepath.Join(os.TempDir(), "JED Simulador"))
}

func clear() {
	if runtime.GOOS == "windows" {
		c := exec.Command("cmd", "/c", "cls")
		c.Stdout = os.Stdout
		_ = c.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func paint(code, text string) string {
	if !colorEnabled {
		return text
	}
	return code + text + ansiReset
}

func accent(text string) string   { return paint(ansiCyan+ansiBold, text) }
func strong(text string) string   { return paint(ansiBold+ansiWhite, text) }
func muted(text string) string    { return paint(ansiDim, text) }
func positive(text string) string { return paint(ansiGreen+ansiBold, text) }
func warning(text string) string  { return paint(ansiYellow+ansiBold, text) }
func negative(text string) string { return paint(ansiRed+ansiBold, text) }
func info(text string) string     { return paint(ansiBlue+ansiBold, text) }
func playful(text string) string  { return paint(ansiMagenta+ansiBold, text) }

func centerPlain(text string, width int) string {
	r := []rune(text)
	if len(r) >= width {
		return text
	}
	left := (width - len(r)) / 2
	return strings.Repeat(" ", left) + text + strings.Repeat(" ", width-len(r)-left)
}

func rule(ch string) string { return strings.Repeat(ch, uiWidth-2) }

func header(title string) {
	clear()
	fmt.Println(accent("╔" + rule("═") + "╗"))
	brand := "JED  •  EMPREENDEDORISMO EM JOGO"
	fmt.Println(accent("║") + playful(centerPlain(brand, uiWidth-2)) + accent("║"))
	fmt.Println(accent("╠" + rule("═") + "╣"))
	fmt.Println(accent("║") + strong(centerPlain(title, uiWidth-2)) + accent("║"))
	fmt.Println(accent("╚" + rule("═") + "╝"))
}

func section(title string) {
	fmt.Println()
	label := " " + strings.ToUpper(title) + " "
	remaining := uiWidth - len([]rune(label)) - 2
	if remaining < 0 {
		remaining = 0
	}
	fmt.Println(accent("┌─") + accent(label) + accent(strings.Repeat("─", remaining)) + accent("┐"))
}

func sectionEnd() { fmt.Println(accent("└" + strings.Repeat("─", uiWidth-2) + "┘")) }

func kv(label, value string) {
	w := 33
	rr := []rune(label)
	if len(rr) > w {
		label = string(rr[:w])
	}
	fmt.Printf("  %-33s %s\n", label, value)
}

func menuLine(key, title, desc string) {
	left := fmt.Sprintf("[%s] %-21s", key, strings.ToUpper(title))
	fmt.Printf("  %s %s\n", accent(left), muted(desc))
}

func bar(value, max float64, width int) string {
	if max <= 0 {
		max = 1
	}
	ratio := clamp(value/max, 0, 1)
	full := int(math.Round(ratio * float64(width)))
	return strings.Repeat("█", full) + strings.Repeat("░", width-full)
}

func reputationVisual(v float64) string {
	b := bar(v, 100, 18)
	if v >= 70 {
		return positive(b + fmt.Sprintf(" %.0f/100", v))
	}
	if v >= 45 {
		return warning(b + fmt.Sprintf(" %.0f/100", v))
	}
	return negative(b + fmt.Sprintf(" %.0f/100", v))
}

func cashVisual(v float64) string {
	if v < 0 {
		return negative(money(v))
	}
	if v < 1000 {
		return warning(money(v))
	}
	return positive(money(v))
}

func statusBadge(text, kind string) string {
	x := "● " + strings.ToUpper(text)
	switch kind {
	case "good":
		return positive(x)
	case "warn":
		return warning(x)
	case "bad":
		return negative(x)
	default:
		return info(x)
	}
}

func readLine() string {
	s, _ := input.ReadString('\n')
	return strings.TrimSpace(s)
}

func pause() { fmt.Print("\n" + muted("[ENTER] continuar") + " "); _, _ = input.ReadString('\n') }

func money(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	s := fmt.Sprintf("%.2f", v)
	parts := strings.Split(s, ".")
	ip := parts[0]
	for i := len(ip) - 3; i > 0; i -= 3 {
		ip = ip[:i] + "." + ip[i:]
	}
	res := "R$ " + ip + "," + parts[1]
	if neg {
		return "-" + res
	}
	return res
}

func pct(v float64) string { return strings.Replace(fmt.Sprintf("%.2f%%", v), ".", ",", 1) }

func parseFloatBR(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, "R$", ""))
	if strings.Contains(s, ",") {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	}
	return strconv.ParseFloat(s, 64)
}

func setHelp(topic string) { helpTopic = topic }

func showHelp(topic string) {
	header("AJUDA  •  " + strings.ToUpper(strings.ReplaceAll(topic, "_", " ")))
	section("Entenda o conceito")
	switch topic {
	case "capital_giro":
		fmt.Println("Capital de giro é o dinheiro que mantém a empresa funcionando enquanto despesas vencem e vendas ainda não foram recebidas. Uma empresa pode ter lucro e, mesmo assim, ficar sem caixa.")
	case "lean_canvas":
		fmt.Println("O Lean Canvas organiza hipóteses do negócio: problema, solução, oferta de valor, vantagem diferencial, segmento de clientes, métricas, canais, estrutura de custos e fontes de receitas. No simulador, várias dessas escolhas alteram alcance, conversão e recorrência.")
	case "cac":
		fmt.Println("CAC é o Custo de Aquisição de Cliente. Exemplo: R$ 200 de marketing / 10 novos clientes = CAC de R$ 20. Compare o CAC com a margem gerada pelo cliente.")
	case "estoque":
		fmt.Println("Estoque é dinheiro transformado em mercadoria ou insumos. Estoque demais imobiliza capital e pode gerar desperdício; estoque de menos provoca vendas perdidas.")
	case "fornecedores":
		fmt.Println("Fornecedores diferem em preço, prazo de entrega, confiabilidade e prazo de pagamento. Comprar mais barato pode significar esperar mais ou correr maior risco de atraso.")
	case "caixa":
		fmt.Println("Caixa mostra dinheiro disponível agora. Receita e lucro não são a mesma coisa que caixa: cartão pode ser recebido depois e compras a prazo podem vencer em outra semana.")
	case "conversao":
		fmt.Println("Conversão é a parcela das pessoas alcançadas que se tornam clientes. Preço, reputação, concorrência, promoção e proposta de valor influenciam essa taxa.")
	case "recorrencia":
		fmt.Println("Recorrência mede clientes que voltam a comprar. Negócios de assinatura ou serviços frequentes dependem especialmente desse indicador.")
	case "ponto_equilibrio":
		fmt.Println("Ponto de equilíbrio é a quantidade aproximada de vendas necessária para cobrir os custos fixos e financeiros da semana, dada a margem de contribuição de cada venda.")
	case "dificuldade":
		fmt.Println("INICIANTE: menor volatilidade, menos eventos negativos, fornecedores mais confiáveis e menor desperdício. INTERMEDIÁRIO: parâmetros-padrão. AVANÇADO: mercado mais volátil, riscos maiores e fornecedores menos previsíveis.")
	case "tutorial":
		fmt.Println("O tutorial ensina capital de giro, margem, CAC, estoque, caixa e validação de hipóteses com exemplos simples. Ele não altera nenhuma empresa salva.")
	default:
		fmt.Println("JED Simulador — ajuda rápida\n\n[V] volta/cancela uma etapa.\n[H] abre ajuda contextual.\n\nConceitos principais: capital de giro, Lean Canvas, conversão, recorrência, CAC, ponto de equilíbrio, estoque, fornecedores e fluxo de caixa.")
	}
	sectionEnd()
	fmt.Println("\n" + info("DICA") + "  Use [H] sempre que encontrar um termo desconhecido.")
	pause()
}

func askText(prompt string, def string, allowBack bool) string {
	for {
		if allowBack {
			fmt.Println("(Digite V para VOLTAR | H para AJUDA)")
		} else {
			fmt.Println("(Digite H para AJUDA)")
		}
		if def != "" {
			fmt.Printf("%s [%s]: ", prompt, def)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		v := readLine()
		if strings.EqualFold(v, "h") {
			showHelp(helpTopic)
			continue
		}
		if allowBack && strings.EqualFold(v, "v") {
			return back
		}
		if v == "" && def != "" {
			return def
		}
		if v != "" {
			return v
		}
		fmt.Println("Preencha o campo.")
	}
}

func askFloat(prompt string, def float64, hasDef bool, min float64, max *float64, allowBack bool) (float64, bool) {
	for {
		if allowBack {
			fmt.Println("(Digite V para VOLTAR | H para AJUDA)")
		} else {
			fmt.Println("(Digite H para AJUDA)")
		}
		if hasDef {
			fmt.Printf("%s [%.2f]: ", prompt, def)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		s := readLine()
		if strings.EqualFold(s, "h") {
			showHelp(helpTopic)
			continue
		}
		if allowBack && strings.EqualFold(s, "v") {
			return 0, true
		}
		if s == "" && hasDef {
			return def, false
		}
		n, err := parseFloatBR(s)
		if err == nil && n >= min && (max == nil || n <= *max) {
			return n, false
		}
		fmt.Println("Valor inválido. Tente novamente ou digite V para voltar.")
	}
}

func askInt(prompt string, def int, hasDef bool, min int, max *int, allowBack bool) (int, bool) {
	for {
		if allowBack {
			fmt.Println("(Digite V para VOLTAR | H para AJUDA)")
		} else {
			fmt.Println("(Digite H para AJUDA)")
		}
		if hasDef {
			fmt.Printf("%s [%d]: ", prompt, def)
		} else {
			fmt.Printf("%s: ", prompt)
		}
		s := readLine()
		if strings.EqualFold(s, "h") {
			showHelp(helpTopic)
			continue
		}
		if allowBack && strings.EqualFold(s, "v") {
			return 0, true
		}
		if s == "" && hasDef {
			return def, false
		}
		n, err := strconv.Atoi(s)
		if err == nil && n >= min && (max == nil || n <= *max) {
			return n, false
		}
		fmt.Println("Número inválido. Tente novamente ou digite V para voltar.")
	}
}

func askOption(prompt string, options map[string]string, order []string, def string, allowBack bool) string {
	for {
		fmt.Println(prompt)
		for _, k := range order {
			fmt.Printf("  %s %s\n", accent("["+k+"]"), options[k])
		}
		if allowBack {
			fmt.Println("  " + accent("[V]") + " VOLTAR")
		}
		fmt.Println("  " + accent("[H]") + " AJUDA")
		fmt.Print("> ")
		v := strings.ToLower(readLine())
		if v == "h" {
			showHelp(helpTopic)
			continue
		}
		if allowBack && v == "v" {
			return back
		}
		if v == "" && def != "" {
			v = def
		}
		if _, ok := options[v]; ok {
			return v
		}
		fmt.Println("Opção inválida.")
	}
}

func askMultiple(prompt string, options map[string]string, order []string, defaults []string, allowBack bool) ([]string, bool) {
	for {
		fmt.Println(prompt)
		for _, k := range order {
			fmt.Printf("  %s %s\n", accent("["+k+"]"), options[k])
		}
		if allowBack {
			fmt.Println("  " + accent("[V]") + " VOLTAR")
		}
		fmt.Println("  " + accent("[H]") + " AJUDA")
		fmt.Print("Digite os números separados por vírgula: ")
		s := strings.ToLower(readLine())
		if s == "h" {
			showHelp(helpTopic)
			continue
		}
		if allowBack && s == "v" {
			return nil, true
		}
		if s == "" && len(defaults) > 0 {
			return defaults, false
		}
		chunks := strings.Split(s, ",")
		seen := map[string]bool{}
		vals := []string{}
		valid := true
		for _, c := range chunks {
			c = strings.TrimSpace(c)
			if _, ok := options[c]; !ok {
				valid = false
				break
			}
			if !seen[c] {
				seen[c] = true
				vals = append(vals, c)
			}
		}
		if valid && len(vals) > 0 {
			return vals, false
		}
		fmt.Println("Escolha inválida.")
	}
}

func slug(s string) string {
	r := strings.ToLower(strings.TrimSpace(s))
	repl := map[string]string{"á": "a", "à": "a", "â": "a", "ã": "a", "é": "e", "ê": "e", "í": "i", "ó": "o", "ô": "o", "õ": "o", "ú": "u", "ç": "c"}
	for a, b := range repl {
		r = strings.ReplaceAll(r, a, b)
	}
	var out []rune
	dash := false
	for _, ch := range r {
		if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-' {
			out = append(out, ch)
			dash = false
		} else if !dash {
			out = append(out, '-')
			dash = true
		}
	}
	r = strings.Trim(string(out), "-")
	if r == "" {
		return "item"
	}
	return r
}

func saveJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0644); err != nil {
		return err
	}
	// Rename within the same directory is effectively atomic on normal local filesystems.
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func newLocalID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := crand.Read(buf); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(buf)
}

func nowStamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func prepareCompanyIdentity(e *Empresa) {
	now := nowStamp()
	if e.LocalID == "" {
		e.LocalID = newLocalID("emp")
	}
	if e.CreatedAt == "" {
		e.CreatedAt = now
	}
	e.UpdatedAt = now
	if e.Revision < 1 {
		e.Revision = 1
	}
	if e.SyncState == "" {
		e.SyncState = "local"
	}
}

func touchCompany(e *Empresa) {
	prepareCompanyIdentity(e)
	e.UpdatedAt = nowStamp()
	e.Revision++
	e.SyncState = "local-modified"
}

func backupCompany(e *Empresa, label string) (string, error) {
	prepareCompanyIdentity(e)
	name := fmt.Sprintf("%s_%s_S%02d_%s.json",
		slug(e.Responsavel), slug(e.Nome), e.Semana, slug(label))
	p := filepath.Join(backupsDir(), name)
	return p, saveJSON(p, e)
}

func loadJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

// ---------- Catálogo ----------

func loadCatalog() (Catalogo, error) {
	var b []byte
	external := filepath.Join(appDir(), "catalogo_negocios.json")
	if x, err := os.ReadFile(external); err == nil {
		b = x
	} else {
		x, err := fs.ReadFile(embeddedFS, "catalogo_negocios.json")
		if err != nil {
			return Catalogo{}, err
		}
		b = x
	}
	var c Catalogo
	err := json.Unmarshal(b, &c)
	return c, err
}

func loadSupplyCatalog() (CatalogoInsumos, error) {
	var raw []byte
	p := filepath.Join(appDir(), "catalogo_insumos.json")
	if b, err := os.ReadFile(p); err == nil {
		raw = b
	} else {
		b, err2 := fs.ReadFile(embeddedFS, "catalogo_insumos.json")
		if err2 != nil {
			return CatalogoInsumos{}, err2
		}
		raw = b
	}
	var c CatalogoInsumos
	if err := json.Unmarshal(raw, &c); err != nil {
		return CatalogoInsumos{}, err
	}
	return c, nil
}

// Pure catalog helpers moved to internal/core in W1.5.

// ---------- Canvas ----------

// ---------- Motor ----------

// ---------- Indicadores / score ----------
// Aggregate analytics moved to internal/core in W1.5.

// ---------- Persistência / Tutor ----------

func companyPath(e *Empresa) string {
	name := slug(e.Responsavel) + "_" + slug(e.Nome) + ".json"
	if e.TurmaID != "" {
		return filepath.Join(classesDir(), e.TurmaID, "empresas", name)
	}
	return filepath.Join(dataDir(), "empresas", name)
}
func saveCompany(e *Empresa) (string, error) {
	e.VersaoDados = version
	touchCompany(e)
	p := companyPath(e)
	return p, saveJSON(p, e)
}
func loadCompany(path string) (*Empresa, error) {
	var e Empresa
	if err := loadJSON(path, &e); err != nil {
		return nil, err
	}
	prepareCompanyIdentity(&e)
	if e.Persona.Nome == "" {
		e.Persona = Persona{Nome: "Cliente principal", Demografia: "A definir", Rotinas: "A definir", Objetivos: "A definir", Desafios: "A definir", Motivadores: "A definir", Objecoes: "A definir", Citacoes: "A definir", PalavrasChave: "A definir"}
	}
	if e.FerramentasDigitais == nil {
		e.FerramentasDigitais = []string{}
	}
	if e.CanaisDigitais == nil {
		e.CanaisDigitais = []string{}
	}
	if e.Reputacao == 0 {
		e.Reputacao = 50
	}
	if e.ReputacaoImportancia == 0 {
		e.ReputacaoImportancia = 1
	}
	if e.DeliveryAfinidade == 0 {
		e.DeliveryAfinidade = 1
	}
	if e.DuracaoSemanas == 0 {
		e.DuracaoSemanas = 24
	}
	e.Dificuldade = normalizeDifficulty(e.Dificuldade)
	if e.InvestimentosIniciais == nil {
		e.InvestimentosIniciais = map[string]float64{}
	}
	if !e.UsaInsumos && e.UsaEstoque && e.ModeloID != "" {
		if cat, err := loadCatalog(); err == nil {
			if m, ok := findModel(cat, e.ModeloID); ok {
				if sc, err2 := loadSupplyCatalog(); err2 == nil {
					p := supplyProfileFor(m, sc)
					if len(p.Insumos) > 0 {
						// Migração conservadora: mantém o valor do estoque antigo e o converte proporcionalmente para os novos insumos.
						baseVal := 0.0
						for _, sp := range p.Insumos {
							baseVal += sp.EstoqueInicial * sp.CustoBase
						}
						factor := 1.0
						if baseVal > 0 && e.EstoqueValor > 0 {
							factor = e.EstoqueValor / baseVal
						}
						e.Insumos = []InsumoEstoque{}
						for _, sp := range p.Insumos {
							e.Insumos = append(e.Insumos, InsumoEstoque{sp.ID, sp.Nome, sp.Unidade, sp.EstoqueInicial * factor, sp.CustoBase, sp.CustoBase, sp.ConsumoPorVenda, sp.ValidadeSemanas, sp.PerdaSemanal, sp.Critico})
						}
						e.UsaInsumos = true
						e.EstoqueUnidades = 0
						e.EstoqueValor = stockValueInputs(&e)
					}
				}
			}
		}
	}
	for i := range e.Insumos {
		if e.Insumos[i].CustoReferencia <= 0 {
			e.Insumos[i].CustoReferencia = e.Insumos[i].CustoMedio
		}
	}
	e.VersaoDados = version
	return &e, nil
}
func saveScenario(c Cenario) (string, error) {
	now := nowStamp()
	if c.LocalID == "" {
		c.LocalID = newLocalID("cen")
	}
	if c.CreatedAt == "" {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	c.Revision++
	if c.Revision < 1 {
		c.Revision = 1
	}
	p := filepath.Join(scenariosDir(), slug(c.Nome)+".json")
	return p, saveJSON(p, c)
}
func listScenarios() []Cenario {
	files, _ := filepath.Glob(filepath.Join(scenariosDir(), "*.json"))
	out := []Cenario{}
	for _, p := range files {
		var c Cenario
		if loadJSON(p, &c) == nil {
			if c.Dificuldade == "" {
				c.Dificuldade = "intermediario"
			}
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nome < out[j].Nome })
	return out
}
func saveClass(t Turma) (string, error) {
	now := nowStamp()
	if t.LocalID == "" {
		t.LocalID = newLocalID("tur")
	}
	if t.CreatedAt == "" {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	t.Revision++
	if t.Revision < 1 {
		t.Revision = 1
	}
	dir := filepath.Join(classesDir(), t.ID)
	_ = os.MkdirAll(filepath.Join(dir, "empresas"), 0755)
	p := filepath.Join(dir, "turma.json")
	return p, saveJSON(p, t)
}
func listClasses() []Turma {
	paths, _ := filepath.Glob(filepath.Join(classesDir(), "*", "turma.json"))
	out := []Turma{}
	for _, p := range paths {
		var t Turma
		if loadJSON(p, &t) == nil {
			out = append(out, t)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Nome < out[j].Nome })
	return out
}
func loadClassCompanies(id string) []*Empresa {
	paths, _ := filepath.Glob(filepath.Join(classesDir(), id, "empresas", "*.json"))
	out := []*Empresa{}
	for _, p := range paths {
		if e, err := loadCompany(p); err == nil {
			out = append(out, e)
		}
	}
	return out
}

func finalReport(e *Empresa) (string, string, Score, error) {
	_ = ensureDirs()
	i := indicators(e)
	sc := score(e)
	stem := slug(e.Responsavel) + "_" + slug(e.Nome)
	txt := filepath.Join(reportsDir(), "relatorio_"+stem+".txt")
	csvp := filepath.Join(reportsDir(), "relatorio_"+stem+".csv")
	var b strings.Builder
	fmt.Fprintf(&b, "JED — RELATÓRIO FINAL\n%s\n", strings.Repeat("=", 70))
	fmt.Fprintf(&b, "Responsável/equipe: %s\nEmpresa: %s\nTurma: %s\nSetor: %s\nTipo: %s\nEspecialidade: %s\nCenário: %s\nDificuldade: %s\nSemanas concluídas: %d/%d\n\n", e.Responsavel, e.Nome, func() string {
		if e.TurmaID == "" {
			return "Individual"
		}
		return e.TurmaID
	}(), e.Setor, e.TipoNegocio, e.Especialidade, e.Cenario, strings.ToUpper(normalizeDifficulty(e.Dificuldade)), e.Semana, e.DuracaoSemanas)
	if i != nil {
		fmt.Fprintf(&b, "INDICADORES\nReceita acumulada: %.2f\nResultado acumulado: %.2f\nCaixa final: %.2f\nContas a receber: %.2f\nContas a pagar: %.2f\nClientes novos: %d\nClientes ativos: %d\nConversão acumulada (%%): %.2f\nCAC aproximado: %.2f\nTicket médio: %.2f\nVendas perdidas: %d\n\n", i.Receita, i.Resultado, i.Caixa, i.AReceber, i.APagar, i.Novos, i.ClientesAtivos, i.ConversaoAcumuladaPct, i.CACAprox, i.TicketMedio, i.Perdidas)
	}
	fmt.Fprintf(&b, "ÍNDICE PEDAGÓGICO\nTotal: %.1f/100\nSustentabilidade financeira: %.1f/25\nMercado e clientes: %.1f/20\nOperação: %.1f/15\nHipóteses e aprendizado: %.1f/25\nGestão: %.1f/15\n\nOBSERVAÇÕES\n", sc.Total, sc.Financeiro, sc.Mercado, sc.Operacao, sc.Hipoteses, sc.Gestao)
	for _, o := range sc.Observacoes {
		fmt.Fprintf(&b, "- %s\n", o)
	}
	fmt.Fprintln(&b, "\nNota: o índice pedagógico é uma referência para discussão e não substitui avaliação qualitativa.")
	if err := os.WriteFile(txt, []byte(b.String()), 0644); err != nil {
		return "", "", sc, err
	}
	f, err := os.Create(csvp)
	if err != nil {
		return "", "", sc, err
	}
	w := csv.NewWriter(f)
	_ = w.Write([]string{"responsavel", "empresa", "turma", "setor", "tipo_negocio", "especialidade", "dificuldade", "semanas", "receita", "resultado", "caixa", "clientes_ativos", "conversao_pct", "cac", "pontuacao"})
	if i == nil {
		i = &Indicadores{Caixa: e.Caixa}
	}
	_ = w.Write([]string{e.Responsavel, e.Nome, e.TurmaID, e.Setor, e.TipoNegocio, e.Especialidade, normalizeDifficulty(e.Dificuldade), strconv.Itoa(e.Semana), fmt.Sprintf("%.2f", i.Receita), fmt.Sprintf("%.2f", i.Resultado), fmt.Sprintf("%.2f", i.Caixa), strconv.Itoa(i.ClientesAtivos), fmt.Sprintf("%.2f", i.ConversaoAcumuladaPct), fmt.Sprintf("%.2f", i.CACAprox), fmt.Sprintf("%.1f", sc.Total)})
	w.Flush()
	_ = f.Close()
	return txt, csvp, sc, w.Error()
}

func compareClass(id string) (string, [][]string, error) {
	es := loadClassCompanies(id)
	type row struct {
		e *Empresa
		s Score
		i *Indicadores
	}
	rows := []row{}
	for _, e := range es {
		rows = append(rows, row{e, score(e), indicators(e)})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].s.Total > rows[j].s.Total })
	out := [][]string{{"responsavel", "empresa", "modelo", "semanas", "pontuacao", "resultado", "caixa", "receita", "clientes_ativos", "conversao_pct", "cac", "vendas_perdidas"}}
	for _, r := range rows {
		i := r.i
		if i == nil {
			i = &Indicadores{Caixa: r.e.Caixa}
		}
		out = append(out, []string{r.e.Responsavel, r.e.Nome, r.e.Especialidade, strconv.Itoa(r.e.Semana), fmt.Sprintf("%.1f", r.s.Total), fmt.Sprintf("%.2f", i.Resultado), fmt.Sprintf("%.2f", i.Caixa), fmt.Sprintf("%.2f", i.Receita), strconv.Itoa(i.ClientesAtivos), fmt.Sprintf("%.2f", i.ConversaoAcumuladaPct), fmt.Sprintf("%.2f", i.CACAprox), strconv.Itoa(i.Perdidas)})
	}
	p := filepath.Join(reportsDir(), "comparativo_"+slug(id)+".csv")
	f, err := os.Create(p)
	if err != nil {
		return "", nil, err
	}
	w := csv.NewWriter(f)
	_ = w.WriteAll(out)
	w.Flush()
	_ = f.Close()
	return p, out, w.Error()
}

// ---------- Wizard ----------

type Context struct {
	TurmaID, Responsavel string
	Cenario              Cenario
}
type Finance struct{ Capital, Emprestimo, Juros float64 }
type Operation struct {
	Operacao, Imovel, Local, DeliveryTipo string
	Aluguel                               float64
	Delivery                              bool
	Funcionarios                          int
	Salario                               float64
}
type Commercial struct {
	Preco, Custo, Marketing, TaxaCartao float64
	Meios                               []string
	PrazoCartao, PrazoFornecedor        int
	ConcNivel                           string
	ConcIndice                          float64
}

func selectClass() (*Turma, bool) {
	ts := listClasses()
	if len(ts) == 0 {
		fmt.Println("\nNenhuma turma cadastrada.")
		pause()
		return nil, true
	}
	header("SELECIONAR TURMA")
	ops := map[string]string{}
	ord := []string{}
	for i, t := range ts {
		k := strconv.Itoa(i + 1)
		ops[k] = t.Nome
		ord = append(ord, k)
	}
	o := askOption("Escolha a turma:", ops, ord, "", true)
	if o == back {
		return nil, true
	}
	t := ts[atoi(o)-1]
	return &t, false
}
func atoi(s string) int { n, _ := strconv.Atoi(s); return n }

func wizardContext() (Context, bool) {
	for {
		header("NOVO EMPREENDIMENTO — ETAPA 1/7 — CONTEXTO")
		mode := askOption("A simulação será:", map[string]string{"1": "Individual", "2": "Vinculada a uma turma"}, []string{"1", "2"}, "1", true)
		if mode == back {
			return Context{}, true
		}
		ctx := Context{}
		if mode == "2" {
			t, b := selectClass()
			if b {
				continue
			}
			ctx.TurmaID = t.ID
			ctx.Cenario = t.Cenario
		} else {
			ops := map[string]string{}
			ord := []string{}
			for i, c := range cenariosBase {
				k := strconv.Itoa(i + 1)
				ops[k] = c.Nome
				ord = append(ord, k)
			}
			x := askOption("\nCenário-base:", ops, ord, "1", true)
			if x == back {
				continue
			}
			ctx.Cenario = cenariosBase[atoi(x)-1]
			d := askOption("\nDuração:", map[string]string{"1": "12 semanas", "2": "24 semanas", "3": "48 semanas"}, []string{"1", "2", "3"}, "2", true)
			if d == back {
				continue
			}
			ctx.Cenario.Duracao = map[string]int{"1": 12, "2": 24, "3": 48}[d]
		}
		if ctx.Cenario.Dificuldade == "" {
			setHelp("dificuldade")
			dif := askOption("\nNível de dificuldade:", map[string]string{"1": "Iniciante", "2": "Intermediário", "3": "Avançado"}, []string{"1", "2", "3"}, "2", true)
			if dif == back {
				continue
			}
			ctx.Cenario.Dificuldade = map[string]string{"1": "iniciante", "2": "intermediario", "3": "avancado"}[dif]
		}
		setHelp("geral")
		r := askText("\nNome do aluno ou da equipe", "", true)
		if r == back {
			continue
		}
		ctx.Responsavel = r
		fmt.Printf("\nResponsável/equipe: %s\nTurma: %s\nCenário: %s\nDificuldade: %s\nDuração: %d semanas\n", r, func() string {
			if ctx.TurmaID == "" {
				return "Individual"
			}
			return ctx.TurmaID
		}(), ctx.Cenario.Nome, strings.ToUpper(ctx.Cenario.Dificuldade), ctx.Cenario.Duracao)
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return Context{}, true
		}
		if a == "r" {
			continue
		}
		return ctx, false
	}
}

func customModel() (Modelo, bool) {
	for {
		header("NEGÓCIO PERSONALIZADO")
		sn := askText("Setor", "", true)
		if sn == back {
			return Modelo{}, true
		}
		tn := askText("Tipo de negócio", "", true)
		if tn == back {
			return Modelo{}, true
		}
		esp := askText("Especialidade / descrição", "", true)
		if esp == back {
			return Modelo{}, true
		}
		price, b := askFloat("Preço médio de referência", 50, true, .01, nil, true)
		if b {
			return Modelo{}, true
		}
		cost, b := askFloat("Custo variável médio por venda", 20, true, 0, nil, true)
		if b {
			return Modelo{}, true
		}
		reach, b := askInt("Alcance-base semanal", 250, true, 1, nil, true)
		if b {
			return Modelo{}, true
		}
		mx := 100.0
		conv, b := askFloat("Conversão-base (%)", 12, true, .1, &mx, true)
		if b {
			return Modelo{}, true
		}
		rec, b := askFloat("Recorrência-base (%)", 10, true, 0, &mx, true)
		if b {
			return Modelo{}, true
		}
		cap, b := askInt("Capacidade-base semanal", 50, true, 1, nil, true)
		if b {
			return Modelo{}, true
		}
		u := askOption("\nUtiliza estoque unitário?", map[string]string{"s": "Sim", "n": "Não"}, []string{"s", "n"}, "n", true)
		if u == back {
			return Modelo{}, true
		}
		stock := u == "s"
		stock0 := 0
		per := 0.0
		if stock {
			stock0, b = askInt("Estoque inicial sugerido", 100, true, 0, nil, true)
			if b {
				return Modelo{}, true
			}
			mx50 := 50.0
			per, b = askFloat("Perecibilidade semanal estimada (%)", 0, true, 0, &mx50, true)
			if b {
				return Modelo{}, true
			}
		}
		funcs, b := askInt("Funcionários sugeridos", 1, true, 0, nil, true)
		if b {
			return Modelo{}, true
		}
		rent, b := askFloat("Aluguel mensal sugerido", 1500, true, 0, nil, true)
		if b {
			return Modelo{}, true
		}
		equip, b := askFloat("Equipamentos sugeridos", 4000, true, 0, nil, true)
		if b {
			return Modelo{}, true
		}
		reg := askOption("\nNegócio regulamentado, com custo adicional de conformidade?", map[string]string{"s": "Sim", "n": "Não"}, []string{"s", "n"}, "n", true)
		if reg == back {
			return Modelo{}, true
		}
		regcost := 0.0
		if reg == "s" {
			regcost, b = askFloat("Custo regulatório mensal estimado", 300, true, 0, nil, true)
			if b {
				return Modelo{}, true
			}
		}
		return Modelo{ID: "personalizado", Nome: esp, PrecoRef: price, CustoUnitario: cost, AlcanceBase: reach, ConversaoBase: conv / 100, RecorrenciaBase: rec / 100, CapacidadeBase: cap, Estoque: stock, Funcionarios: funcs, Aluguel: rent, Equipamentos: equip, EstoqueInicialUn: stock0, Sazonalidade: .08, PerecibilidadeSemanal: per / 100, Regulamentado: reg == "s", CustoRegulatorioMensal: regcost, ReputacaoImportancia: 1, DeliveryAfinidade: 1, SetorID: "personalizado", SetorNome: sn, TipoID: "personalizado", TipoNome: tn}, false
	}
}

func selectModel(cat Catalogo) (Modelo, bool) {
	for {
		header("NOVO EMPREENDIMENTO — ETAPA 2/7 — NEGÓCIO")
		fmt.Printf("Catálogo atual: %d especialidades.\n\n", catalogCount(cat))
		for i, s := range cat.Setores {
			fmt.Printf("[%d] %s\n", i+1, s.Nome)
		}
		fmt.Println("[P] Negócio personalizado\n[V] VOLTAR")
		fmt.Print("\nSetor: ")
		s := strings.ToLower(readLine())
		if s == "v" {
			return Modelo{}, true
		}
		if s == "p" {
			return customModel()
		}
		si := atoi(s) - 1
		if si < 0 || si >= len(cat.Setores) {
			fmt.Println("Setor inválido.")
			pause()
			continue
		}
		sec := cat.Setores[si]
		for {
			header("SETOR — " + sec.Nome)
			for i, t := range sec.Tipos {
				fmt.Printf("[%d] %s\n", i+1, t.Nome)
			}
			fmt.Println("[V] VOLTAR")
			fmt.Print("\nTipo de negócio: ")
			x := strings.ToLower(readLine())
			if x == "v" {
				break
			}
			ti := atoi(x) - 1
			if ti < 0 || ti >= len(sec.Tipos) {
				fmt.Println("Tipo inválido.")
				pause()
				continue
			}
			typ := sec.Tipos[ti]
			for {
				header(sec.Nome + " > " + typ.Nome)
				for i, m := range typ.Especialidades {
					tags := []string{}
					if m.Regulamentado {
						tags = append(tags, "REGULADO")
					}
					if m.PerecibilidadeSemanal >= .08 {
						tags = append(tags, "PERECÍVEL")
					}
					extra := ""
					if len(tags) > 0 {
						extra = " [" + strings.Join(tags, " | ") + "]"
					}
					fmt.Printf("[%d] %s%s\n", i+1, m.Nome, extra)
				}
				fmt.Println("[V] VOLTAR")
				fmt.Print("\nEspecialidade: ")
				z := strings.ToLower(readLine())
				if z == "v" {
					break
				}
				mi := atoi(z) - 1
				if mi < 0 || mi >= len(typ.Especialidades) {
					fmt.Println("Especialidade inválida.")
					pause()
					continue
				}
				m := typ.Especialidades[mi]
				m.SetorID = sec.ID
				m.SetorNome = sec.Nome
				m.TipoID = typ.ID
				m.TipoNome = typ.Nome
				header("CONFIRMAR MODELO DE NEGÓCIO")
				fmt.Printf("Setor............................. %s\nTipo.............................. %s\nEspecialidade..................... %s\nPreço de referência............... %s\nCusto variável.................... %s\nCapacidade-base................... %d/semana\nUsa estoque....................... %v\n", m.SetorNome, m.TipoNome, m.Nome, money(m.PrecoRef), money(m.CustoUnitario), m.CapacidadeBase, m.Estoque)
				if m.Estoque {
					fmt.Printf("Perecibilidade semanal............. %.1f%%\n", m.PerecibilidadeSemanal*100)
				}
				fmt.Printf("Regulamentado...................... %v\n", m.Regulamentado)
				c := askOption("\nUsar este modelo?", map[string]string{"s": "Sim", "n": "Não"}, []string{"s", "n"}, "s", true)
				if c == "s" {
					return m, false
				}
			}
		}
	}
}

func wizardModel(cat Catalogo) (Modelo, string, bool) {
	for {
		m, b := selectModel(cat)
		if b {
			return Modelo{}, "", true
		}
		header("NOVO EMPREENDIMENTO — ETAPA 2/7 — IDENTIFICAÇÃO")
		name := askText("Nome da empresa", m.Nome, true)
		if name == back {
			continue
		}
		fmt.Printf("\nSetor: %s\nTipo: %s\nEspecialidade: %s\nEmpresa: %s\n", m.SetorNome, m.TipoNome, m.Nome, name)
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return Modelo{}, "", true
		}
		if a == "r" {
			continue
		}
		return m, name, false
	}
}

func wizardFinance() (Finance, bool) {
	setHelp("capital_giro")
	for {
		header("NOVO EMPREENDIMENTO — ETAPA 3/7 — CAPITAL")
		cap, b := askFloat("Capital próprio", 20000, true, 0, nil, true)
		if b {
			return Finance{}, true
		}
		has := askOption("\nHaverá empréstimo inicial?", map[string]string{"s": "Sim", "n": "Não"}, []string{"s", "n"}, "n", true)
		if has == back {
			return Finance{}, true
		}
		loan, rate := 0.0, 0.0
		if has == "s" {
			loan, b = askFloat("Valor do empréstimo", 5000, true, 0, nil, true)
			if b {
				continue
			}
			rate, b = askFloat("Juros mensais simplificados (%)", 2, true, 0, nil, true)
			if b {
				continue
			}
		}
		fmt.Printf("\nCapital próprio: %s\nEmpréstimo: %s\nCapital total: %s\n", money(cap), money(loan), money(cap+loan))
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return Finance{}, true
		}
		if a == "r" {
			continue
		}
		return Finance{cap, loan, rate}, false
	}
}

func wizardOperation(m Modelo) (Operation, bool) {
	for {
		header("NOVO EMPREENDIMENTO — ETAPA 4/7 — OPERAÇÃO")
		op := askOption("Como a empresa opera?", map[string]string{"1": "Somente presencial", "2": "Somente digital", "3": "Híbrida"}, []string{"1", "2", "3"}, "1", true)
		if op == back {
			return Operation{}, true
		}
		o := Operation{}
		o.Operacao = map[string]string{"1": "fisica", "2": "digital", "3": "hibrida"}[op]
		if o.Operacao == "digital" {
			o.Imovel = "sem_imovel"
			o.Local = "media"
		} else {
			im := askOption("\nSituação do imóvel:", map[string]string{"1": "Próprio", "2": "Alugado", "3": "Cedido", "4": "Coworking / compartilhado"}, []string{"1", "2", "3", "4"}, "2", true)
			if im == back {
				continue
			}
			o.Imovel = map[string]string{"1": "proprio", "2": "alugado", "3": "cedido", "4": "coworking"}[im]
			var b bool
			if o.Imovel == "alugado" || o.Imovel == "coworking" {
				o.Aluguel, b = askFloat("Custo mensal do espaço", m.Aluguel, true, 0, nil, true)
				if b {
					continue
				}
			}
			loc := askOption("\nQualidade da localização:", map[string]string{"1": "Baixa", "2": "Média", "3": "Alta"}, []string{"1", "2", "3"}, "2", true)
			if loc == back {
				continue
			}
			o.Local = map[string]string{"1": "baixa", "2": "media", "3": "alta"}[loc]
		}
		dl := askOption("\nOferece delivery/entrega?", map[string]string{"s": "Sim", "n": "Não"}, []string{"s", "n"}, "n", true)
		if dl == back {
			continue
		}
		o.Delivery = dl == "s"
		o.DeliveryTipo = "nenhum"
		if o.Delivery {
			d := askOption("\nTipo de delivery:", map[string]string{"1": "Próprio", "2": "Plataforma", "3": "Misto"}, []string{"1", "2", "3"}, "2", true)
			if d == back {
				continue
			}
			o.DeliveryTipo = map[string]string{"1": "proprio", "2": "plataforma", "3": "misto"}[d]
		}
		f, b := askInt("\nQuantidade de funcionários", m.Funcionarios, true, 0, nil, true)
		if b {
			continue
		}
		o.Funcionarios = f
		if f > 0 {
			o.Salario, b = askFloat("Salário médio mensal", 1800, true, 0, nil, true)
			if b {
				continue
			}
		}
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return Operation{}, true
		}
		if a == "r" {
			continue
		}
		return o, false
	}
}

func wizardCommercial(m Modelo, ctx Context) (Commercial, bool) {
	for {
		header("NOVO EMPREENDIMENTO — ETAPA 5/7 — COMERCIAL")
		c := Commercial{}
		var b bool
		c.Preco, b = askFloat("Preço médio de venda", m.PrecoRef, true, .01, nil, true)
		if b {
			return Commercial{}, true
		}
		c.Custo, b = askFloat("Custo variável médio por venda/unidade", m.CustoUnitario, true, 0, nil, true)
		if b {
			return Commercial{}, true
		}
		c.Marketing, b = askFloat("Marketing semanal", 150, true, 0, nil, true)
		if b {
			return Commercial{}, true
		}
		vals, bm := askMultiple("\nMeios de pagamento:", map[string]string{"1": "Dinheiro", "2": "PIX", "3": "Cartão"}, []string{"1", "2", "3"}, []string{"2", "3"}, true)
		if bm {
			return Commercial{}, true
		}
		for _, x := range vals {
			c.Meios = append(c.Meios, map[string]string{"1": "dinheiro", "2": "pix", "3": "cartao"}[x])
		}
		if contains(c.Meios, "cartao") {
			c.TaxaCartao, b = askFloat("Taxa média do cartão (%)", 2.5, true, 0, nil, true)
			if b {
				continue
			}
			p := askOption("\nPrazo de recebimento do cartão:", map[string]string{"0": "Imediato", "1": "1 semana", "2": "2 semanas"}, []string{"0", "1", "2"}, "1", true)
			if p == back {
				continue
			}
			c.PrazoCartao = atoi(p)
		}
		p := askOption("\nPrazo máximo oferecido por fornecedor:", map[string]string{"0": "À vista", "1": "1 semana", "2": "2 semanas"}, []string{"0", "1", "2"}, "2", true)
		if p == back {
			continue
		}
		c.PrazoFornecedor = atoi(p)
		if ctx.TurmaID != "" && ctx.Cenario.ConcorrenciaIndice > 0 {
			c.ConcNivel = ctx.Cenario.ConcorrenciaNivel
			c.ConcIndice = ctx.Cenario.ConcorrenciaIndice
			fmt.Printf("\nConcorrência definida pelo tutor: %s (%.2f)\n", strings.ToUpper(c.ConcNivel), c.ConcIndice)
		} else {
			x := askOption("\nConcorrência inicial:", map[string]string{"1": "Baixa", "2": "Média", "3": "Alta"}, []string{"1", "2", "3"}, "2", true)
			if x == back {
				continue
			}
			c.ConcNivel = map[string]string{"1": "baixa", "2": "media", "3": "alta"}[x]
			c.ConcIndice = map[string]float64{"1": .80, "2": 1, "3": 1.20}[x]
		}
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return Commercial{}, true
		}
		if a == "r" {
			continue
		}
		return c, false
	}
}

func fillCanvas() (LeanCanvas, bool) {
	setHelp("lean_canvas")
	for {
		header("NOVO EMPREENDIMENTO — ETAPA 6/7 — LEAN CANVAS")
		c := LeanCanvas{}
		c.Problema = askText("1. PROBLEMA — quais problemas você quer resolver?", "", true)
		if c.Problema == back {
			return LeanCanvas{}, true
		}
		seg, b := askMultiple("\n2. SEGMENTOS DE CLIENTES:", map[string]string{"1": "Sensíveis a preço", "2": "Orientados à qualidade", "3": "Buscam conveniência", "4": "Fortemente digitais", "5": "Empresas / B2B", "6": "Público geral"}, []string{"1", "2", "3", "4", "5", "6"}, []string{"6"}, true)
		if b {
			return LeanCanvas{}, true
		}
		sm := map[string]string{"1": "sensivel_preco", "2": "qualidade", "3": "conveniencia", "4": "digital", "5": "b2b", "6": "geral"}
		for _, x := range seg {
			c.Segmentos = append(c.Segmentos, sm[x])
		}
		p := askOption("\n3. OFERTA DE VALOR:", map[string]string{"1": "Menor preço", "2": "Qualidade", "3": "Rapidez", "4": "Conveniência", "5": "Personalização", "6": "Atendimento", "7": "Exclusividade"}, []string{"1", "2", "3", "4", "5", "6", "7"}, "2", true)
		if p == back {
			return LeanCanvas{}, true
		}
		c.PropostaValor = map[string]string{"1": "menor_preco", "2": "qualidade", "3": "rapidez", "4": "conveniencia", "5": "personalizacao", "6": "atendimento", "7": "exclusividade"}[p]
		c.Solucao = askText("\n4. SOLUÇÃO — o que você oferece?", "", true)
		if c.Solucao == back {
			return LeanCanvas{}, true
		}
		ch, b := askMultiple("\n5. CANAIS:", map[string]string{"1": "Redes sociais", "2": "Busca na internet", "3": "Indicação", "4": "Loja física", "5": "Marketplace", "6": "Prospecção direta"}, []string{"1", "2", "3", "4", "5", "6"}, []string{"1"}, true)
		if b {
			return LeanCanvas{}, true
		}
		cm := map[string]string{"1": "redes_sociais", "2": "busca_online", "3": "indicacao", "4": "loja_fisica", "5": "marketplace", "6": "prospeccao"}
		for _, x := range ch {
			c.Canais = append(c.Canais, cm[x])
		}
		r := askOption("\n6. FONTES DE RECEITA:", map[string]string{"1": "Venda única", "2": "Projeto", "3": "Assinatura", "4": "Comissão"}, []string{"1", "2", "3", "4"}, "1", true)
		if r == back {
			return LeanCanvas{}, true
		}
		c.ReceitaModelo = map[string]string{"1": "venda_unica", "2": "projeto", "3": "assinatura", "4": "comissao"}[r]
		c.CustosNotas = askText("\n7. ESTRUTURA DE CUSTOS — principais custos:", "", true)
		if c.CustosNotas == back {
			return LeanCanvas{}, true
		}
		met, b := askMultiple("\n8. MÉTRICAS-CHAVE:", map[string]string{"1": "Vendas", "2": "Faturamento", "3": "Margem", "4": "Caixa", "5": "Recorrência", "6": "Conversão", "7": "CAC"}, []string{"1", "2", "3", "4", "5", "6", "7"}, []string{"1", "2", "4", "6"}, true)
		if b {
			return LeanCanvas{}, true
		}
		mm := map[string]string{"1": "vendas", "2": "faturamento", "3": "margem", "4": "caixa", "5": "recorrencia", "6": "conversao", "7": "cac"}
		for _, x := range met {
			c.Metricas = append(c.Metricas, mm[x])
		}
		c.Vantagem = askText("\n9. VANTAGEM INJUSTA — o que é difícil de copiar?", "", true)
		if c.Vantagem == back {
			return LeanCanvas{}, true
		}
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return LeanCanvas{}, true
		}
		if a == "r" {
			continue
		}
		return c, false
	}
}

func fillHypotheses(price float64, m Modelo) (Hipoteses, bool) {
	header("NOVO EMPREENDIMENTO — ETAPA 7/7 — HIPÓTESES E SEMANA ZERO")
	v, b := askFloat("Vendas esperadas por semana", 30, true, 0, nil, true)
	if b {
		return Hipoteses{}, true
	}
	rev, b := askFloat("Faturamento esperado por semana", v*price, true, 0, nil, true)
	if b {
		return Hipoteses{}, true
	}
	mx := 100.0
	conv, b := askFloat("Conversão esperada (%)", m.ConversaoBase*100, true, 0, &mx, true)
	if b {
		return Hipoteses{}, true
	}
	rec, b := askFloat("Recorrência esperada (%)", m.RecorrenciaBase*100, true, 0, &mx, true)
	if b {
		return Hipoteses{}, true
	}
	p := askOption("\nEspera resultado positivo na maioria das semanas?", map[string]string{"s": "Sim", "n": "Não"}, []string{"s", "n"}, "s", true)
	if p == back {
		return Hipoteses{}, true
	}
	return Hipoteses{v, rev, p == "s", conv, rec}, false
}

func weekZero(total float64, m Modelo, cost float64, imovel string, profile PerfilInsumos) (map[string]float64, int, float64, []InsumoEstoque, float64, bool) {
	setHelp("capital_giro")
	for {
		header("SEMANA ZERO — DISTRIBUIÇÃO DO CAPITAL")
		fmt.Printf("Capital disponível: %s\n\n", money(total))
		reform, b := askFloat("Reforma / adaptação", 0, true, 0, nil, true)
		if b {
			return nil, 0, 0, nil, 0, true
		}
		equip, b := askFloat("Equipamentos e mobiliário", m.Equipamentos, true, 0, nil, true)
		if b {
			return nil, 0, 0, nil, 0, true
		}
		lic, b := askFloat("Licenças / abertura / documentação", 500, true, 0, nil, true)
		if b {
			return nil, 0, 0, nil, 0, true
		}
		deposit := 0.0
		if imovel == "alugado" {
			deposit, b = askFloat("Caução do aluguel", m.Aluguel, true, 0, nil, true)
			if b {
				return nil, 0, 0, nil, 0, true
			}
		}
		stockN := 0
		stockVal := 0.0
		inputs := []InsumoEstoque{}
		if len(profile.Insumos) > 0 {
			fmt.Println("\nESTOQUE INICIAL DE INSUMOS")
			level := askOption("Nível inicial:", map[string]string{"1": "75% do sugerido", "2": "100% do sugerido", "3": "125% do sugerido", "4": "150% do sugerido"}, []string{"1", "2", "3", "4"}, "2", true)
			if level == back {
				return nil, 0, 0, nil, 0, true
			}
			factor := map[string]float64{"1": .75, "2": 1, "3": 1.25, "4": 1.5}[level]
			recipeCost := 0.0
			for _, sp := range profile.Insumos {
				recipeCost += sp.CustoBase * sp.ConsumoPorVenda
			}
			costScale := 1.0
			if recipeCost > 0 && cost > 0 {
				costScale = cost / recipeCost
			}
			for _, sp := range profile.Insumos {
				q := sp.EstoqueInicial * factor
				scaledCost := sp.CustoBase * costScale
				v := q * scaledCost
				stockVal += v
				inputs = append(inputs, InsumoEstoque{sp.ID, sp.Nome, sp.Unidade, q, scaledCost, scaledCost, sp.ConsumoPorVenda, sp.ValidadeSemanas, sp.PerdaSemanal, sp.Critico})
				fmt.Printf("- %-32s %8.1f %-8s %12s\n", sp.Nome, q, sp.Unidade, money(v))
			}
			fmt.Printf("Total dos insumos iniciais: %s\n", money(stockVal))
		} else if m.Estoque {
			stockN, b = askInt("Unidades de estoque inicial", m.EstoqueInicialUn, true, 0, nil, true)
			if b {
				return nil, 0, 0, nil, 0, true
			}
			stockVal = float64(stockN) * cost
			fmt.Printf("Valor do estoque inicial: %s\n", money(stockVal))
		}
		launch, b := askFloat("Marketing de lançamento", 600, true, 0, nil, true)
		if b {
			return nil, 0, 0, nil, 0, true
		}
		spent := reform + equip + lic + deposit + stockVal + launch
		left := total - spent
		fmt.Printf("\n%s\nTotal comprometido................. %s\nCapital de giro restante........... %s\n", strings.Repeat("-", 76), money(spent), money(left))
		if left < 0 {
			fmt.Println("\nCapital insuficiente. Refaça a Semana Zero.")
			pause()
			continue
		}
		a := askOption("\nO que deseja fazer?", map[string]string{"c": "CONTINUAR", "r": "REFAZER ESTA ETAPA"}, []string{"c", "r"}, "c", true)
		if a == back {
			return nil, 0, 0, nil, 0, true
		}
		if a == "r" {
			continue
		}
		inv := map[string]float64{"reforma": reform, "equipamentos": equip, "equipamentos_referencia": m.Equipamentos, "licencas": lic, "caucao": deposit, "estoque_inicial": stockVal, "marketing_lancamento": launch, "capital_giro_inicial": left}
		return inv, stockN, stockVal, inputs, left, false
	}
}

func createCompany(cat Catalogo, sc CatalogoInsumos) *Empresa {
	var ctx Context
	var m Modelo
	var name string
	var fin Finance
	var op Operation
	var com Commercial
	var canv LeanCanvas
	var hyp Hipoteses
	var inv map[string]float64
	stockN := 0
	stockVal := 0.0
	inputs := []InsumoEstoque{}
	cash := 0.0
	step := 0
	for step < 7 {
		switch step {
		case 0:
			x, b := wizardContext()
			if b {
				return nil
			}
			ctx = x
		case 1:
			x, n, b := wizardModel(cat)
			if b {
				step--
				continue
			}
			m = x
			name = n
		case 2:
			x, b := wizardFinance()
			if b {
				step--
				continue
			}
			fin = x
		case 3:
			x, b := wizardOperation(m)
			if b {
				step--
				continue
			}
			op = x
		case 4:
			x, b := wizardCommercial(m, ctx)
			if b {
				step--
				continue
			}
			com = x
		case 5:
			x, b := fillCanvas()
			if b {
				step--
				continue
			}
			canv = x
		case 6:
			x, b := fillHypotheses(com.Preco, m)
			if b {
				step--
				continue
			}
			hyp = x
			profile := supplyProfileFor(m, sc)
			z, n, v, ins, c, b2 := weekZero(fin.Capital+fin.Emprestimo, m, com.Custo, op.Imovel, profile)
			if b2 {
				step--
				continue
			}
			inv = z
			stockN = n
			stockVal = v
			inputs = ins
			cash = c
		}
		step++
	}
	cen := ctx.Cenario
	if cen.Duracao == 0 {
		cen.Duracao = 24
	}
	e := &Empresa{Nome: name, Responsavel: ctx.Responsavel, TurmaID: ctx.TurmaID, ModeloBase: m.Nome, Categoria: m.SetorID, ModeloID: m.ID, Setor: m.SetorNome, TipoNegocio: m.TipoNome, Especialidade: m.Nome, PerecibilidadeSemanal: m.PerecibilidadeSemanal, Regulamentado: m.Regulamentado, CustoRegulatorioMensal: m.CustoRegulatorioMensal, ReputacaoImportancia: m.ReputacaoImportancia, DeliveryAfinidade: m.DeliveryAfinidade, ModeloDigital: m.ModeloDigital, CapitalProprio: fin.Capital, EmprestimoInicial: fin.Emprestimo, JurosMensal: fin.Juros, Divida: fin.Emprestimo, Caixa: cash, Preco: com.Preco, PrecoReferencia: m.PrecoRef, CustoUnitario: com.Custo, AlcanceBase: m.AlcanceBase, ConversaoBase: m.ConversaoBase, RecorrenciaBase: m.RecorrenciaBase, CapacidadeBase: m.CapacidadeBase, SazonalidadeAmplitude: m.Sazonalidade, SazonalidadeFase: m.Fase, UsaEstoque: m.Estoque, EstoqueUnidades: stockN, EstoqueValor: stockVal, CustoMedioEstoque: com.Custo, UsaInsumos: len(inputs) > 0, Insumos: inputs, Operacao: op.Operacao, Imovel: op.Imovel, AluguelMensal: op.Aluguel, QualidadeLocalizacao: op.Local, Delivery: op.Delivery, DeliveryTipo: op.DeliveryTipo, Funcionarios: op.Funcionarios, SalarioMedio: op.Salario, MarketingSemanal: com.Marketing, MeiosPagamento: com.Meios, TaxaCartao: com.TaxaCartao, PrazoCartaoSemanas: com.PrazoCartao, PrazoFornecedorMax: com.PrazoFornecedor, Cenario: cen.Nome, CenarioAlcance: cen.Alcance, CenarioConversao: cen.Conversao, CenarioOscilacao: cen.Oscilacao, CenarioEventoNegExtra: cen.EventoNegativoExtra, DuracaoSemanas: cen.Duracao, ConcorrenciaNivel: com.ConcNivel, ConcorrenciaIndice: com.ConcIndice, Reputacao: 50, Dificuldade: normalizeDifficulty(cen.Dificuldade), Canvas: canv, Hipoteses: hyp, InvestimentosIniciais: inv, VersaoDados: version}
	return e
}

// ---------- Telas da empresa ----------

func showCanvas(e *Empresa) {
	header("LEAN CANVAS — " + e.Nome)
	c := e.Canvas
	fmt.Printf("\nPROBLEMA\n%s\n\nSEGMENTOS\n%s\n\nPROPOSTA DE VALOR\n%s\n\nSOLUÇÃO\n%s\n\nCANAIS\n%s\n\nMODELO DE RECEITA\n%s\n\nESTRUTURA DE CUSTOS\n%s\n\nMÉTRICAS\n%s\n\nVANTAGEM INJUSTA\n%s\n", c.Problema, strings.Join(c.Segmentos, ", "), c.PropostaValor, c.Solucao, strings.Join(c.Canais, ", "), c.ReceitaModelo, c.CustosNotas, strings.Join(c.Metricas, ", "), c.Vantagem)
	ex := canvasExplanations(e)
	if len(ex) > 0 {
		fmt.Println("\nEFEITOS ATIVOS")
		for _, x := range ex {
			fmt.Println("-", x)
		}
	}
	pause()
}
func reviseCanvas(e *Empresa) {
	c, b := fillCanvas()
	if b {
		return
	}
	e.Canvas = c
	e.RevisoesCanvas++
	fmt.Println("\nLean Canvas revisado. A alteração valerá na próxima semana.")
	pause()
}
func showWeekZero(e *Empresa) {
	header("SEMANA ZERO — " + e.Nome)
	fmt.Printf("Capital próprio..................... %s\nEmpréstimo.......................... %s\n%s\n", money(e.CapitalProprio), money(e.EmprestimoInicial), strings.Repeat("-", 76))
	keys := []string{"reforma", "equipamentos", "licencas", "caucao", "estoque_inicial", "marketing_lancamento"}
	for _, k := range keys {
		if v, ok := e.InvestimentosIniciais[k]; ok {
			fmt.Printf("%-36s %s\n", strings.Title(strings.ReplaceAll(k, "_", " ")), money(v))
		}
	}
	fmt.Println(strings.Repeat("-", 76))
	fmt.Printf("Capital de giro inicial............. %s\nCaixa atual......................... %s\n", money(e.InvestimentosIniciais["capital_giro_inicial"]), money(e.Caixa))
	pause()
}
func changePrice(e *Empresa) {
	header("ALTERAR PREÇO")
	fmt.Println("Preço atual:", money(e.Preco))
	v, b := askFloat("Novo preço", e.Preco, true, .01, nil, true)
	if !b {
		e.Preco = v
	}
}
func changeMarketing(e *Empresa) {
	header("ALTERAR MARKETING")
	fmt.Println("Marketing atual:", money(e.MarketingSemanal))
	v, b := askFloat("Novo investimento semanal", e.MarketingSemanal, true, 0, nil, true)
	if !b {
		e.MarketingSemanal = v
	}
}
func changeStaff(e *Empresa) {
	header("ALTERAR EQUIPE")
	q, b := askInt("Quantidade de funcionários", e.Funcionarios, true, 0, nil, true)
	if b {
		return
	}
	sal := e.SalarioMedio
	if q > 0 {
		sal, b = askFloat("Salário médio mensal", func() float64 {
			if sal > 0 {
				return sal
			}
			return 1800
		}(), true, 0, nil, true)
		if b {
			return
		}
	}
	e.Funcionarios = q
	e.SalarioMedio = sal
}
func setPromotion(e *Empresa) {
	header("PROMOÇÃO — PRÓXIMA SEMANA")
	fmt.Println("Preço normal:", money(e.Preco))
	mx := 35.0
	d, b := askFloat("Desconto (%)", 0, true, 0, &mx, true)
	if b {
		return
	}
	e.PromocaoDesconto = d
	fmt.Println("Preço promocional:", money(e.Preco*(1-d/100)))
	pause()
}
func showSupplies(e *Empresa) {
	header("INSUMOS E FORNECEDORES")
	if !e.UsaInsumos {
		fmt.Printf("Estoque genérico: %d un. | valor %s\n", e.EstoqueUnidades, money(e.EstoqueValor))
		pause()
		return
	}
	fmt.Println("\nINSUMOS")
	for i, x := range e.Insumos {
		coverage := "n/d"
		if x.ConsumoPorVenda > 0 {
			coverage = fmt.Sprintf("%.0f vendas", math.Floor(x.Quantidade/x.ConsumoPorVenda))
		}
		fmt.Printf("[%d] %-28s %8.1f %-7s | custo %9s | cobre %s\n", i+1, x.Nome, x.Quantidade, x.Unidade, money(x.CustoMedio), coverage)
	}
	fmt.Printf("\nValor total em insumos: %s\n", money(stockValueInputs(e)))
	if len(e.PedidosInsumos) > 0 {
		fmt.Println("\nPEDIDOS A CAMINHO")
		for _, o := range e.PedidosInsumos {
			fmt.Printf("Semana %d | %-26s %7.1f | %s\n", o.SemanaEntrega, o.InsumoNome, o.Quantidade, o.Fornecedor)
		}
	}
	sc, err := loadSupplyCatalog()
	if err == nil {
		fmt.Println("\nFORNECEDORES DISPONÍVEIS")
		for _, f := range sc.Fornecedores {
			fmt.Printf("- %s: preço x%.2f, entrega %d sem., confiabilidade %.0f%%\n", f.Nome, f.MultiplicadorPreco, f.PrazoEntregaSemanas, f.Confiabilidade*100)
		}
	}
	pause()
}

func stockAction(e *Empresa) {
	setHelp("fornecedores")
	if !e.UsaInsumos {
		if !e.UsaEstoque {
			fmt.Println("\nEste negócio não utiliza estoque unitário.")
			pause()
			return
		}
		header("COMPRAR ESTOQUE")
		q, b := askInt("Quantidade a comprar", 0, true, 0, nil, true)
		if b {
			return
		}
		ops := map[string]string{"0": "À vista"}
		ord := []string{"0"}
		for i := 1; i <= e.PrazoFornecedorMax; i++ {
			k := strconv.Itoa(i)
			ops[k] = fmt.Sprintf("Em %d semana(s)", i)
			ord = append(ord, k)
		}
		p := askOption("Condição de pagamento:", ops, ord, "0", true)
		if p == back {
			return
		}
		ok, c, msg := buyStock(e, q, atoi(p))
		if ok {
			fmt.Printf("\nCompra realizada: %s. Valor: %s\n", msg, money(c))
		} else {
			fmt.Printf("\nCompra não realizada: %s. Valor: %s\n", msg, money(c))
		}
		pause()
		return
	}
	header("COMPRAR INSUMOS")
	if len(e.Insumos) == 0 {
		fmt.Println("Nenhum insumo configurado.")
		pause()
		return
	}
	ops := map[string]string{}
	ord := []string{}
	for i, x := range e.Insumos {
		k := strconv.Itoa(i + 1)
		ops[k] = fmt.Sprintf("%s — atual %.1f %s", x.Nome, x.Quantidade, x.Unidade)
		ord = append(ord, k)
	}
	choice := askOption("Escolha o insumo:", ops, ord, "", true)
	if choice == back {
		return
	}
	idx := atoi(choice) - 1
	if idx < 0 || idx >= len(e.Insumos) {
		return
	}
	q, b := askFloat(fmt.Sprintf("Quantidade (%s)", e.Insumos[idx].Unidade), 0, false, .01, nil, true)
	if b {
		return
	}
	sc, err := loadSupplyCatalog()
	if err != nil {
		fmt.Println("Erro ao carregar fornecedores:", err)
		pause()
		return
	}
	fops := map[string]string{}
	ford := []string{}
	for i, f := range sc.Fornecedores {
		k := strconv.Itoa(i + 1)
		fops[k] = fmt.Sprintf("%s — preço x%.2f; entrega %d sem.; confiança %.0f%%", f.Nome, f.MultiplicadorPreco, f.PrazoEntregaSemanas, f.Confiabilidade*100)
		ford = append(ford, k)
	}
	fc := askOption("Fornecedor:", fops, ford, "2", true)
	if fc == back {
		return
	}
	fi := atoi(fc) - 1
	if fi < 0 || fi >= len(sc.Fornecedores) {
		return
	}
	f := sc.Fornecedores[fi]
	maxTerm := minInt(e.PrazoFornecedorMax, f.PrazoPagamentoMax)
	tops := map[string]string{"0": "À vista"}
	tord := []string{"0"}
	for i := 1; i <= maxTerm; i++ {
		k := strconv.Itoa(i)
		tops[k] = fmt.Sprintf("Pagar em %d semana(s)", i)
		tord = append(tord, k)
	}
	term := askOption("Pagamento:", tops, tord, "0", true)
	if term == back {
		return
	}
	ok, c, msg := placeInputOrder(e, e.Insumos[idx].ID, q, f, atoi(term))
	if ok {
		fmt.Printf("\nPedido realizado. Valor: %s\n%s\n", money(c), msg)
	} else {
		fmt.Printf("\nPedido não realizado: %s. Valor: %s\n", msg, money(c))
	}
	pause()
}

func showAccounts(e *Empresa) {
	setHelp("caixa")
	header("FINANCEIRO  •  CONTAS")
	section("Resumo")
	kv("Caixa agora", cashVisual(e.Caixa))
	kv("Total a receber", positive(money(sumAccounts(e.ContasReceber))))
	kv("Total a pagar", func() string {
		if sumAccounts(e.ContasPagar) > e.Caixa {
			return warning(money(sumAccounts(e.ContasPagar)))
		}
		return money(sumAccounts(e.ContasPagar))
	}())
	sectionEnd()

	section("A receber")
	if len(e.ContasReceber) == 0 {
		fmt.Println("  " + muted("Nenhum recebimento futuro registrado."))
	} else {
		for _, x := range e.ContasReceber {
			fmt.Printf("  %s  semana %-2d  %s\n", positive(money(x.Valor)), x.Semana, muted(x.Descricao))
		}
	}
	sectionEnd()

	section("A pagar")
	if len(e.ContasPagar) == 0 {
		fmt.Println("  " + muted("Nenhum pagamento futuro registrado."))
	} else {
		for _, x := range e.ContasPagar {
			fmt.Printf("  %s  semana %-2d  %s\n", warning(money(x.Valor)), x.Semana, muted(x.Descricao))
		}
	}
	sectionEnd()
	fmt.Println("\n" + info("H") + "  ajuda: lucro, receita e caixa não são a mesma coisa.")
	pause()
}
func showIndicators(e *Empresa) {
	setHelp("cac")
	header("INDICADORES  •  " + strings.ToUpper(e.Nome))
	i := indicators(e)
	if i == nil {
		fmt.Println("\n" + muted("Ainda não há semanas concluídas."))
		pause()
		return
	}

	section("Mercado e clientes")
	kv("Alcance acumulado", strong(strconv.Itoa(i.Alcance)))
	kv("Clientes novos", strong(strconv.Itoa(i.Novos)))
	kv("Clientes ativos", strong(strconv.Itoa(i.ClientesAtivos)))
	kv("Conversão", info(pct(i.ConversaoAcumuladaPct)))
	kv("CAC aproximado", money(i.CACAprox))
	sectionEnd()

	section("Vendas")
	kv("Vendas realizadas", strong(strconv.Itoa(i.Vendas)))
	lostKind := "good"
	if i.Perdidas > 0 {
		lostKind = "warn"
	}
	kv("Vendas perdidas", statusBadge(strconv.Itoa(i.Perdidas), lostKind))
	kv("Ticket médio", money(i.TicketMedio))
	sectionEnd()

	section("Dinheiro")
	kv("Receita acumulada", positive(money(i.Receita)))
	if i.Resultado >= 0 {
		kv("Resultado acumulado", positive(money(i.Resultado)))
	} else {
		kv("Resultado acumulado", negative(money(i.Resultado)))
	}
	kv("Caixa", cashVisual(i.Caixa))
	sectionEnd()
	pause()
}
func showWeekReport(r Registro) {
	setHelp("ponto_equilibrio")
	header(fmt.Sprintf("FECHAMENTO DA SEMANA %d", r.Semana))
	if r.Evento != "" {
		section("Aconteceu esta semana")
		fmt.Println("  " + playful("◆ ") + r.Evento)
		sectionEnd()
	}

	section("Clientes")
	kv("Alcance", strconv.Itoa(r.Alcance))
	kv("Conversão", info(pct(r.ConversaoObservadaPct)))
	kv("Novos clientes", strong(strconv.Itoa(r.NovosClientes)))
	kv("Clientes recorrentes", strconv.Itoa(r.ClientesRecorrentes))
	kv("Base ativa", strong(strconv.Itoa(r.ClientesAtivosFinal)))
	sectionEnd()

	section("Operação")
	kv("Demanda", strconv.Itoa(r.Demanda))
	kv("Capacidade", strconv.Itoa(r.Capacidade))
	kv("Vendas", positive(strconv.Itoa(r.Vendas)))
	if r.VendasPerdidas > 0 {
		kv("Vendas perdidas", warning(strconv.Itoa(r.VendasPerdidas)))
	} else {
		kv("Vendas perdidas", positive("0"))
	}
	if r.Gargalo != "" {
		kv("Gargalo", warning(strings.ToUpper(r.Gargalo)))
	}
	if r.InsumoLimitante != "" {
		kv("Insumo limitante", warning(r.InsumoLimitante))
	}
	if r.EntregasRecebidas != "" {
		kv("Entregas recebidas", positive(r.EntregasRecebidas))
	}
	sectionEnd()

	section("Economia unitária")
	kv("Preço efetivo", money(r.PrecoEfetivo))
	kv("Ticket médio", money(r.TicketMedio))
	if r.CAC == nil {
		kv("CAC", "n/d")
	} else {
		kv("CAC", money(*r.CAC))
	}
	kv("Margem de contribuição", money(r.MargemContribuicaoUnit)+" / venda")
	if r.PontoEquilibrioVendas == nil {
		kv("Ponto de equilíbrio", "n/d")
	} else {
		kv("Ponto de equilíbrio", fmt.Sprintf("%d vendas", *r.PontoEquilibrioVendas))
	}
	sectionEnd()

	section("Placar financeiro")
	kv("Receita", positive(money(r.Receita)))
	if r.Resultado >= 0 {
		kv("Resultado da semana", positive(money(r.Resultado)))
	} else {
		kv("Resultado da semana", negative(money(r.Resultado)))
	}
	if r.FluxoCaixa >= 0 {
		kv("Fluxo de caixa", positive(money(r.FluxoCaixa)))
	} else {
		kv("Fluxo de caixa", negative(money(r.FluxoCaixa)))
	}
	kv("Caixa final", cashVisual(r.Caixa))
	kv("A receber", money(r.ContasReceber))
	kv("A pagar", money(r.ContasPagar))
	sectionEnd()

	if r.DesperdicioDetalhes != "" || r.DesperdicioUnidades > 0 {
		section("Desperdício")
		if r.DesperdicioDetalhes != "" {
			kv("Perda econômica", negative(money(r.DesperdicioInsumos)))
			fmt.Println("  " + strings.ReplaceAll(r.DesperdicioDetalhes, "\n", "\n  "))
		} else {
			kv("Unidades perdidas", strconv.Itoa(r.DesperdicioUnidades))
			kv("Valor perdido", negative(money(r.DesperdicioValor)))
		}
		sectionEnd()
	}

	if r.Resultado >= 0 && r.Caixa >= 0 {
		fmt.Println("\n" + statusBadge("semana encerrada com resultado e caixa positivos", "good"))
	} else if r.Caixa < 0 {
		fmt.Println("\n" + statusBadge("caixa negativo — reveja prazos e capital de giro", "bad"))
	} else {
		fmt.Println("\n" + statusBadge("resultado negativo — investigue margem, custos e volume", "warn"))
	}
	pause()
}
func showReview(r Review) {
	header("CHECKPOINT  •  HIPÓTESES DE 4 SEMANAS")
	statusPaint := func(x string) string {
		if x == "VALIDADA" || x == "SUPERADA" {
			return positive(x)
		}
		if x == "NÃO VALIDADA" {
			return warning(x)
		}
		return x
	}
	section("O que você previa × o que aconteceu")
	fmt.Printf("  %-18s %14s %14s   %s\n", "Métrica", "Esperado", "Observado", "Status")
	fmt.Println("  " + strings.Repeat("─", 68))
	fmt.Printf("  %-18s %14.2f %14.2f   %s\n", "Vendas", r.MetaVendas, r.MediaVendas, statusPaint(r.StatusVendas))
	fmt.Printf("  %-18s %14s %14s   %s\n", "Faturamento", money(r.MetaReceita), money(r.MediaReceita), statusPaint(r.StatusReceita))
	fmt.Printf("  %-18s %14s %14s   %s\n", "Conversão", pct(r.MetaConversao), pct(r.MediaConversao), statusPaint(r.StatusConversao))
	fmt.Printf("  %-18s %14s %14s   %s\n", "Recorrência", pct(r.MetaRecorrencia), pct(r.MediaRecorrencia), statusPaint(r.StatusRecorrencia))
	sectionEnd()
	fmt.Println("\n" + info("PERGUNTA PARA A EQUIPE") + "  O que os dados sugerem: perseverar, ajustar ou pivotar?")
	pause()
}
func generateReportUI(e *Empresa) {
	txt, csvp, s, err := finalReport(e)
	header("RELATÓRIO FINAL")
	if err != nil {
		fmt.Println("Erro:", err)
		pause()
		return
	}
	fmt.Printf("Pontuação pedagógica: %.1f/100\nTXT: %s\nCSV: %s\n\nFinanceiro........ %.1f/25\nMercado........... %.1f/20\nOperação.......... %.1f/15\nHipóteses......... %.1f/25\nGestão............ %.1f/15\n", s.Total, txt, csvp, s.Financeiro, s.Mercado, s.Operacao, s.Hipoteses, s.Gestao)
	pause()
}

func panel(e *Empresa) {
	for {
		header(fmt.Sprintf("%s  •  SEMANA %d DE %d", strings.ToUpper(e.Nome), e.Semana+1, e.DuracaoSemanas))

		progress := bar(float64(e.Semana), float64(maxInt(e.DuracaoSemanas, 1)), 30)
		section("Seu negócio")
		kv("Equipe", e.Responsavel)
		kv("Atividade", e.Setor+" › "+e.TipoNegocio+" › "+e.Especialidade)
		kv("Dificuldade", strings.ToUpper(normalizeDifficulty(e.Dificuldade)))
		kv("Jornada", accent(progress)+fmt.Sprintf(" %d/%d", e.Semana, e.DuracaoSemanas))
		sectionEnd()

		section("Pulso da empresa")
		kv("CAIXA", cashVisual(e.Caixa))
		kv("CLIENTES ATIVOS", strong(strconv.Itoa(e.ClientesAtivos)))
		kv("REPUTAÇÃO", reputationVisual(e.Reputacao))
		if e.UsaInsumos {
			kv("INSUMOS", fmt.Sprintf("%d itens • %s", len(e.Insumos), money(stockValueInputs(e))))
		} else if e.UsaEstoque {
			kv("ESTOQUE", fmt.Sprintf("%d unidades", e.EstoqueUnidades))
		}
		if e.Caixa < 0 {
			fmt.Println("  " + statusBadge("caixa negativo — atenção à liquidez", "bad"))
		} else if sumAccounts(e.ContasPagar) > e.Caixa+sumAccounts(e.ContasReceber) {
			fmt.Println("  " + statusBadge("compromissos futuros exigem atenção", "warn"))
		} else {
			fmt.Println("  " + statusBadge("operação em andamento", "good"))
		}
		sectionEnd()

		section("Escolha sua próxima decisão")
		menuLine("1", "Preço", "ajustar o preço médio")
		menuLine("2", "Promoção", "criar desconto para a próxima semana")
		menuLine("3", "Compras", "estoque, insumos e fornecedores")
		menuLine("4", "Marketing", "alterar investimento em aquisição")
		menuLine("5", "Equipe", "contratar ou reduzir funcionários")
		menuLine("6", "Lean Canvas", "consultar estratégia atual")
		menuLine("7", "Pivotar", "revisar hipóteses e proposta")
		menuLine("8", "Indicadores", "ver desempenho acumulado")
		menuLine("9", "Financeiro", "contas a receber e a pagar")
		menuLine("10", "Semana Zero", "rever investimentos de abertura")
		menuLine("11", "Fechar semana", "processar mercado e resultados")
		menuLine("12", "Salvar", "salvar agora")
		menuLine("13", "Relatório", "gerar relatório pedagógico")
		menuLine("14", "Insumos", "ver estoque e pedidos em trânsito")
		menuLine("H", "Ajuda", "explicação da tela atual")
		menuLine("V", "Voltar", "menu principal")
		sectionEnd()

		fmt.Print("\nEscolha: ")
		x := strings.ToLower(readLine())
		switch x {
		case "1":
			changePrice(e)
		case "2":
			setPromotion(e)
		case "3":
			stockAction(e)
		case "4":
			changeMarketing(e)
		case "5":
			changeStaff(e)
		case "6":
			showCanvas(e)
		case "7":
			reviseCanvas(e)
		case "8":
			showIndicators(e)
		case "9":
			showAccounts(e)
		case "10":
			showWeekZero(e)
		case "11":
			if e.Semana >= e.DuracaoSemanas {
				fmt.Println("\n" + warning("A simulação já foi concluída."))
				pause()
				continue
			}
			r := processWeek(e)
			showWeekReport(r)
			if rv := reviewHypotheses(e); rv != nil {
				showReview(*rv)
			}
			_, _ = saveCompany(e)
			if e.Semana >= e.DuracaoSemanas {
				generateReportUI(e)
			}
		case "12":
			if p, err := saveCompany(e); err != nil {
				fmt.Println(negative("Erro: ") + err.Error())
			} else {
				fmt.Println("\n" + positive("✓ Empresa salva") + "\n" + muted(p))
			}
			pause()
		case "13":
			generateReportUI(e)
		case "14":
			showSupplies(e)
		case "h":
			setHelp("geral")
			showHelp(helpTopic)
		case "v":
			return
		}
	}
}

// ---------- Tutorial / primeira execução ----------

type AppConfig struct {
	TutorialConcluido bool   `json:"tutorial_concluido"`
	Tema              string `json:"tema"`
}

func configPath() string { return filepath.Join(dataDir(), "config.json") }
func loadConfig() AppConfig {
	var c AppConfig
	_ = loadJSON(configPath(), &c)
	if c.Tema == "" {
		c.Tema = "colorido"
	}
	return c
}
func saveConfig(c AppConfig) { _ = saveJSON(configPath(), c) }

func applyAppearance(c AppConfig) {
	colorEnabled = colorCapable && c.Tema != "sem_cor"
}

func appearanceUI() {
	c := loadConfig()
	applyAppearance(c)
	for {
		header("APARÊNCIA")
		section("Tema visual")
		kv("Tema atual", func() string {
			if colorEnabled {
				return positive("COLORIDO")
			}
			return "SEM CORES"
		}())
		fmt.Println()
		menuLine("1", "Colorido", "cores e destaques quando o terminal permitir")
		menuLine("2", "Sem cores", "mesmo layout, indicado para consoles muito antigos")
		menuLine("V", "Voltar", "retornar ao menu principal")
		sectionEnd()
		fmt.Print("\nEscolha: ")
		x := strings.ToLower(readLine())
		if x == "v" {
			return
		}
		if x == "1" {
			c.Tema = "colorido"
			applyAppearance(c)
			saveConfig(c)
		} else if x == "2" {
			c.Tema = "sem_cor"
			applyAppearance(c)
			saveConfig(c)
		}
	}
}

func tutorialMode() {
	setHelp("tutorial")
	header("MISSÃO 1/6  •  CAPITAL DE GIRO")
	fmt.Println("Você tem R$ 10.000 para abrir uma pequena empresa.")
	fmt.Println()
	x := askOption("Como distribuir o dinheiro?", map[string]string{"1": "R$ 9.000 em equipamentos e R$ 1.000 em caixa", "2": "R$ 5.000 em equipamentos e R$ 5.000 em caixa"}, []string{"1", "2"}, "2", true)
	if x == back {
		return
	}
	if x == "1" {
		fmt.Println("\nVocê ficou com pouco fôlego financeiro. Mesmo uma empresa lucrativa pode ficar sem dinheiro antes de receber as vendas.")
	} else {
		fmt.Println("\nBoa reserva. Os R$ 5.000 restantes funcionam como CAPITAL DE GIRO para pagar despesas do começo da operação.")
	}
	pause()

	header("MISSÃO 2/6  •  PREÇO E MARGEM")
	fmt.Println("Seu produto custa R$ 8 para ser entregue ao cliente.")
	x = askOption("Qual preço parece mais seguro?", map[string]string{"1": "R$ 10", "2": "R$ 18"}, []string{"1", "2"}, "2", true)
	if x == back {
		return
	}
	if x == "1" {
		fmt.Println("\nMargem bruta aproximada: R$ 2. Um preço baixo pode vender mais, mas sobra pouco para pagar aluguel, salários e marketing.")
	} else {
		fmt.Println("\nMargem bruta aproximada: R$ 10. Mas preço maior também pode reduzir a conversão. O objetivo é equilibrar valor percebido, volume e margem.")
	}
	pause()

	header("MISSÃO 3/6  •  CAC")
	fmt.Println("Você gastou R$ 200 em marketing e conquistou 10 novos clientes.\nCAC = R$ 200 / 10 = R$ 20 por novo cliente.\n\nSe cada cliente gera menos de R$ 20 de margem, a aquisição está cara demais.")
	pause()

	header("MISSÃO 4/6  •  ESTOQUE")
	fmt.Println("Comprar estoque demais prende dinheiro e, em alimentos, pode gerar desperdício. Comprar pouco demais provoca vendas perdidas.\n\nNa v0.8, fornecedores, validade e insumos fazem parte dessas decisões.")
	pause()

	header("MISSÃO 5/6  •  LUCRO NÃO É CAIXA")
	fmt.Println("Você vende R$ 1.000 no cartão hoje, mas só recebe na próxima semana. O aluguel vence hoje.\n\nA venda existe e pode gerar lucro, mas o dinheiro ainda não entrou no CAIXA. Por isso capital de giro e prazos importam.")
	pause()

	header("MISSÃO 6/6  •  HIPÓTESES LEAN")
	fmt.Println("Antes de começar, você registra hipóteses: vendas, faturamento, conversão e recorrência. A cada quatro semanas o simulador compara expectativa e realidade.\n\nQuando os dados contradizem a hipótese, a ideia não é 'perder o jogo': é aprender, ajustar ou pivotar.")
	fmt.Println("\nTutorial concluído. Use [H] AJUDA durante o simulador sempre que encontrar um conceito desconhecido.")
	pause()
}

func firstRun() {
	c := loadConfig()
	applyAppearance(c)
	if c.TutorialConcluido {
		return
	}
	header("BEM-VINDO AO JED SIMULADOR")
	fmt.Println(playful("Seu primeiro negócio começa aqui."))
	fmt.Println(muted("Sem instalação, sem Python e sem internet."))
	fmt.Println()
	x := askOption("Primeira vez por aqui?", map[string]string{"1": "COMEÇAR TUTORIAL", "2": "IR DIRETO AO SIMULADOR"}, []string{"1", "2"}, "1", false)
	if x == "1" {
		tutorialMode()
	}
	c.TutorialConcluido = true
	saveConfig(c)
}

// ---------- Laboratório de balanceamento ----------

type LabResult struct {
	ModeloID, Setor, Tipo, Especialidade, Estrategia, Dificuldade                       string
	Execucoes                                                                           int
	SobrevivenciaPct, CaixaMedio, ResultadoMedio, VendasPerdidasMedia, DesperdicioMedio float64
}

func makeLabCompany(m Modelo, profile PerfilInsumos, strategy, difficulty string) *Empresa {
	price := m.PrecoRef
	marketing := 120.0
	staff := m.Funcionarios
	proposal := "qualidade"
	switch strategy {
	case "crescimento":
		price *= .98
		marketing = 350
		staff++
	case "preco_baixo":
		price *= .86
		marketing = 220
		proposal = "menor_preco"
	case "premium":
		price *= 1.20
		marketing = 200
		proposal = "qualidade"
	}
	inputs := []InsumoEstoque{}
	stockVal := 0.0
	for _, sp := range profile.Insumos {
		q := sp.EstoqueInicial * 2
		inputs = append(inputs, InsumoEstoque{sp.ID, sp.Nome, sp.Unidade, q, sp.CustoBase, sp.CustoBase, sp.ConsumoPorVenda, sp.ValidadeSemanas, sp.PerdaSemanal, sp.Critico})
		stockVal += q * sp.CustoBase
	}
	stockN := 0
	if m.Estoque && len(inputs) == 0 {
		stockN = maxInt(m.EstoqueInicialUn*2, m.CapacidadeBase*3)
		stockVal = float64(stockN) * m.CustoUnitario
	}
	startup := m.Equipamentos + stockVal + 500 + m.Aluguel
	capital := startup + math.Max(8000, m.Aluguel*3+float64(staff)*1800)
	cash := capital - startup
	op := "fisica"
	if m.Aluguel == 0 {
		op = "digital"
	}
	delivery := m.DeliveryAfinidade > 1.08
	return &Empresa{Nome: "LAB", Responsavel: "LAB", ModeloBase: m.Nome, Categoria: m.SetorID, ModeloID: m.ID, Setor: m.SetorNome, TipoNegocio: m.TipoNome, Especialidade: m.Nome, PerecibilidadeSemanal: m.PerecibilidadeSemanal, Regulamentado: m.Regulamentado, CustoRegulatorioMensal: m.CustoRegulatorioMensal, ReputacaoImportancia: m.ReputacaoImportancia, DeliveryAfinidade: m.DeliveryAfinidade, ModeloDigital: m.ModeloDigital, Dificuldade: difficulty, CapitalProprio: capital, Caixa: cash, Preco: price, PrecoReferencia: m.PrecoRef, CustoUnitario: m.CustoUnitario, AlcanceBase: m.AlcanceBase, ConversaoBase: m.ConversaoBase, RecorrenciaBase: m.RecorrenciaBase, CapacidadeBase: m.CapacidadeBase, SazonalidadeAmplitude: m.Sazonalidade, SazonalidadeFase: m.Fase, UsaEstoque: m.Estoque, EstoqueUnidades: stockN, EstoqueValor: stockVal, CustoMedioEstoque: m.CustoUnitario, UsaInsumos: len(inputs) > 0, Insumos: inputs, Operacao: op, Imovel: "alugado", AluguelMensal: m.Aluguel, QualidadeLocalizacao: "media", Delivery: delivery, DeliveryTipo: "misto", Funcionarios: staff, SalarioMedio: 1800, MarketingSemanal: marketing, MeiosPagamento: []string{"pix"}, PrazoFornecedorMax: 2, Cenario: "LAB", CenarioAlcance: 1, CenarioConversao: 1, CenarioOscilacao: .08, DuracaoSemanas: 24, ConcorrenciaNivel: "media", ConcorrenciaIndice: 1, Reputacao: 50, Canvas: LeanCanvas{Problema: "lab", Segmentos: []string{"geral"}, PropostaValor: proposal, Solucao: m.Nome, Canais: []string{"redes_sociais"}, ReceitaModelo: "venda_unica"}, Hipoteses: Hipoteses{float64(m.CapacidadeBase) * .6, price * float64(m.CapacidadeBase) * .6, true, m.ConversaoBase * 100, m.RecorrenciaBase * 100}, InvestimentosIniciais: map[string]float64{"equipamentos": m.Equipamentos, "equipamentos_referencia": m.Equipamentos, "capital_giro_inicial": cash}}
}

func labRestock(e *Empresa) {
	if e.UsaInsumos {
		for i := range e.Insumos {
			target := float64(e.CapacidadeBase) * e.Insumos[i].ConsumoPorVenda * 2.2
			if e.Insumos[i].Quantidade < target*.55 {
				q := target - e.Insumos[i].Quantidade
				cost := q * e.Insumos[i].CustoReferencia
				if cost <= e.Caixa {
					oldV := e.Insumos[i].Quantidade * e.Insumos[i].CustoMedio
					e.Caixa -= cost
					e.Insumos[i].Quantidade += q
					e.Insumos[i].CustoMedio = (oldV + cost) / e.Insumos[i].Quantidade
				}
			}
		}
		e.EstoqueValor = stockValueInputs(e)
		return
	}
	if e.UsaEstoque && e.EstoqueUnidades < e.CapacidadeBase*2 {
		q := e.CapacidadeBase*3 - e.EstoqueUnidades
		cost := float64(q) * e.CustoUnitario
		if cost <= e.Caixa {
			e.Caixa -= cost
			e.EstoqueUnidades += q
			e.EstoqueValor += cost
			e.CustoMedioEstoque = e.EstoqueValor / float64(e.EstoqueUnidades)
		}
	}
}

func runBalanceLab(cat Catalogo, sc CatalogoInsumos, runs int, difficulty string) ([]LabResult, string, error) {
	if runs < 1 {
		runs = 1
	}
	strategies := []string{"conservadora", "crescimento", "preco_baixo", "premium"}
	out := []LabResult{}
	for _, setor := range cat.Setores {
		for _, tipo := range setor.Tipos {
			for _, raw := range tipo.Especialidades {
				m := raw
				m.SetorID = setor.ID
				m.SetorNome = setor.Nome
				m.TipoID = tipo.ID
				m.TipoNome = tipo.Nome
				profile := supplyProfileFor(m, sc)
				for _, st := range strategies {
					surv := 0
					cash, result, lost, waste := 0.0, 0.0, 0.0, 0.0
					for r := 0; r < runs; r++ {
						e := makeLabCompany(m, profile, st, difficulty)
						for w := 0; w < 24; w++ {
							labRestock(e)
							rr := processWeek(e)
							lost += float64(rr.VendasPerdidas)
							waste += rr.DesperdicioValor
						}
						ind := indicators(e)
						if e.Caixa+sumAccounts(e.ContasReceber)-sumAccounts(e.ContasPagar) >= 0 {
							surv++
						}
						cash += e.Caixa
						if ind != nil {
							result += ind.Resultado
						}
					}
					out = append(out, LabResult{m.ID, setor.Nome, tipo.Nome, m.Nome, st, difficulty, runs, float64(surv) / float64(runs) * 100, cash / float64(runs), result / float64(runs), lost / float64(runs), waste / float64(runs)})
				}
			}
		}
	}
	p := filepath.Join(reportsDir(), fmt.Sprintf("laboratorio_balanceamento_%s.csv", difficulty))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return nil, "", err
	}
	f, err := os.Create(p)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"modelo_id", "setor", "tipo", "especialidade", "estrategia", "dificuldade", "execucoes", "sobrevivencia_pct", "caixa_medio", "resultado_medio", "vendas_perdidas_media", "desperdicio_medio"})
	for _, x := range out {
		_ = w.Write([]string{x.ModeloID, x.Setor, x.Tipo, x.Especialidade, x.Estrategia, x.Dificuldade, strconv.Itoa(x.Execucoes), fmt.Sprintf("%.2f", x.SobrevivenciaPct), fmt.Sprintf("%.2f", x.CaixaMedio), fmt.Sprintf("%.2f", x.ResultadoMedio), fmt.Sprintf("%.2f", x.VendasPerdidasMedia), fmt.Sprintf("%.2f", x.DesperdicioMedio)})
	}
	return out, p, w.Error()
}

func balanceLabUI() {
	header("MODO TUTOR — LABORATÓRIO DE BALANCEAMENTO")
	setHelp("dificuldade")
	r := askOption("Execuções por estratégia e especialidade:", map[string]string{"1": "10 (rápido)", "2": "25 (recomendado)", "3": "50 (mais estável)"}, []string{"1", "2", "3"}, "2", true)
	if r == back {
		return
	}
	runs := map[string]int{"1": 10, "2": 25, "3": 50}[r]
	d := askOption("Dificuldade:", map[string]string{"1": "Iniciante", "2": "Intermediário", "3": "Avançado"}, []string{"1", "2", "3"}, "2", true)
	if d == back {
		return
	}
	diff := map[string]string{"1": "iniciante", "2": "intermediario", "3": "avancado"}[d]
	cat, err := loadCatalog()
	if err != nil {
		fmt.Println("Erro:", err)
		pause()
		return
	}
	sc, err := loadSupplyCatalog()
	if err != nil {
		fmt.Println("Erro:", err)
		pause()
		return
	}
	fmt.Printf("\nExecutando %d especialidades × 4 estratégias × %d repetições...\n", catalogCount(cat), runs)
	rows, p, err := runBalanceLab(cat, sc, runs, diff)
	if err != nil {
		fmt.Println("Erro:", err)
		pause()
		return
	}
	type agg struct {
		name string
		sum  float64
		n    int
	}
	mm := map[string]*agg{}
	for _, x := range rows {
		a := mm[x.ModeloID]
		if a == nil {
			a = &agg{name: x.Especialidade}
			mm[x.ModeloID] = a
		}
		a.sum += x.SobrevivenciaPct
		a.n++
	}
	list := []agg{}
	for _, a := range mm {
		list = append(list, *a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].sum/float64(list[i].n) < list[j].sum/float64(list[j].n) })
	fmt.Println("\nModelos com menor sobrevivência média:")
	for i := 0; i < minInt(8, len(list)); i++ {
		fmt.Printf("%2d. %-32s %6.1f%%\n", i+1, list[i].name, list[i].sum/float64(list[i].n))
	}
	fmt.Println("\nCSV completo:", p)
	pause()
}

// ---------- Tutor UI ----------

func createScenarioUI() {
	for {
		header("MODO TUTOR — CRIAR CENÁRIO")
		name := askText("Nome do cenário", "", true)
		if name == back {
			return
		}
		ops := map[string]string{}
		ord := []string{}
		for i, c := range cenariosBase {
			k := strconv.Itoa(i + 1)
			ops[k] = c.Nome
			ord = append(ord, k)
		}
		b := askOption("\nBase econômica:", ops, ord, "1", true)
		if b == back {
			return
		}
		c := cenariosBase[atoi(b)-1]
		d := askOption("\nDuração:", map[string]string{"1": "12 semanas", "2": "24 semanas", "3": "48 semanas"}, []string{"1", "2", "3"}, "2", true)
		if d == back {
			return
		}
		c.Duracao = map[string]int{"1": 12, "2": 24, "3": 48}[d]
		co := askOption("\nConcorrência inicial para toda a turma:", map[string]string{"1": "Baixa", "2": "Média", "3": "Alta"}, []string{"1", "2", "3"}, "2", true)
		if co == back {
			return
		}
		c.ConcorrenciaNivel = map[string]string{"1": "baixa", "2": "media", "3": "alta"}[co]
		c.ConcorrenciaIndice = map[string]float64{"1": .8, "2": 1, "3": 1.2}[co]
		setHelp("dificuldade")
		dif := askOption("\nDificuldade da turma:", map[string]string{"1": "Iniciante", "2": "Intermediário", "3": "Avançado"}, []string{"1", "2", "3"}, "2", true)
		if dif == back {
			return
		}
		c.Dificuldade = map[string]string{"1": "iniciante", "2": "intermediario", "3": "avancado"}[dif]
		setHelp("geral")
		obs := askText("\nObservações do tutor", "Cenário criado para atividade em sala", true)
		if obs == back {
			return
		}
		c.Nome = name
		c.Observacoes = obs
		p, err := saveScenario(c)
		if err != nil {
			fmt.Println("Erro:", err)
		} else {
			fmt.Println("\nCenário salvo em:\n" + p)
		}
		pause()
		return
	}
}
func createClassUI() {
	cs := listScenarios()
	if len(cs) == 0 {
		fmt.Println("\nCrie pelo menos um cenário antes de criar uma turma.")
		pause()
		return
	}
	header("MODO TUTOR — CRIAR TURMA")
	ops := map[string]string{}
	ord := []string{}
	for i, c := range cs {
		k := strconv.Itoa(i + 1)
		ops[k] = c.Nome
		ord = append(ord, k)
	}
	o := askOption("Escolha o cenário:", ops, ord, "", true)
	if o == back {
		return
	}
	name := askText("\nNome da turma", "", true)
	if name == back {
		return
	}
	tutor := askText("Nome do tutor", "", true)
	if tutor == back {
		return
	}
	t := Turma{ID: slug(name), Nome: name, Tutor: tutor, Cenario: cs[atoi(o)-1]}
	p, err := saveClass(t)
	if err != nil {
		fmt.Println("Erro:", err)
	} else {
		fmt.Printf("\nTurma criada: %s (%s)\nArquivo: %s\n", t.Nome, t.ID, p)
	}
	pause()
}
func listClassesUI() {
	header("MODO TUTOR — TURMAS")
	ts := listClasses()
	if len(ts) == 0 {
		fmt.Println("\nNenhuma turma criada.")
	} else {
		for _, t := range ts {
			fmt.Printf("- %s | ID: %s | Cenário: %s\n", t.Nome, t.ID, t.Cenario.Nome)
		}
	}
	pause()
}
func compareClassUI() {
	t, b := selectClass()
	if b {
		return
	}
	p, rows, err := compareClass(t.ID)
	header("COMPARATIVO — " + t.Nome)
	if err != nil {
		fmt.Println("Erro:", err)
		pause()
		return
	}
	if len(rows) <= 1 {
		fmt.Println("\nAinda não há empresas salvas nesta turma.")
		pause()
		return
	}
	fmt.Printf("%-3s %-18s %-18s %6s %14s %14s\n", "#", "Equipe", "Empresa", "Pts", "Resultado", "Caixa")
	fmt.Println(strings.Repeat("-", 76))
	for i, r := range rows[1:] {
		res, _ := strconv.ParseFloat(r[5], 64)
		cash, _ := strconv.ParseFloat(r[6], 64)
		fmt.Printf("%-3d %-18.18s %-18.18s %6s %14s %14s\n", i+1, r[0], r[1], r[4], money(res), money(cash))
	}
	fmt.Println("\nCSV completo:", p)
	pause()
}
func tutorMode() {
	for {
		header("MODO TUTOR")
		section("Sala de aula")
		menuLine("1", "Criar cenário", "definir mercado, duração e dificuldade")
		menuLine("2", "Criar turma", "vincular uma turma a um cenário")
		menuLine("3", "Listar turmas", "ver turmas cadastradas")
		menuLine("4", "Comparar empresas", "ranking e CSV da turma")
		menuLine("5", "Balanceamento", "simular automaticamente todas as especialidades")
		menuLine("H", "Ajuda", "orientação sobre o modo tutor")
		menuLine("V", "Voltar", "retornar ao painel inicial")
		sectionEnd()
		fmt.Print("\nEscolha: ")
		x := strings.ToLower(readLine())
		switch x {
		case "1":
			createScenarioUI()
		case "2":
			createClassUI()
		case "3":
			listClassesUI()
		case "4":
			compareClassUI()
		case "5":
			balanceLabUI()
		case "h":
			setHelp("geral")
			showHelp(helpTopic)
		case "v":
			return
		}
	}
}

func loadCompanyUI() *Empresa {
	header("CARREGAR EMPREENDIMENTO")
	p := askText("Caminho do arquivo JSON", "", true)
	if p == back {
		return nil
	}
	e, err := loadCompany(p)
	if err != nil {
		fmt.Println("\nNão foi possível carregar:", err)
		pause()
		return nil
	}
	return e
}

// ---------- Self-test ----------

func selfTest() error {
	cat, err := loadCatalog()
	if err != nil {
		return err
	}
	if catalogCount(cat) < 80 {
		return fmt.Errorf("catálogo pequeno: %d", catalogCount(cat))
	}
	sc, err := loadSupplyCatalog()
	if err != nil {
		return err
	}
	if len(sc.Fornecedores) < 3 {
		return errors.New("fornecedores insuficientes")
	}
	sushi, ok := findModel(cat, "rest_sushi")
	if !ok {
		return errors.New("sushi ausente")
	}
	profile := supplyProfileFor(sushi, sc)
	if len(profile.Insumos) < 5 {
		return errors.New("perfil de sushi incompleto")
	}
	ins := []InsumoEstoque{}
	val := 0.0
	for _, sp := range profile.Insumos {
		ins = append(ins, InsumoEstoque{sp.ID, sp.Nome, sp.Unidade, sp.EstoqueInicial, sp.CustoBase, sp.CustoBase, sp.ConsumoPorVenda, sp.ValidadeSemanas, sp.PerdaSemanal, sp.Critico})
		val += sp.EstoqueInicial * sp.CustoBase
	}
	rng = rand.New(rand.NewSource(99))
	e := &Empresa{Nome: "Teste", Responsavel: "Equipe", Dificuldade: "intermediario", ModeloBase: sushi.Nome, Categoria: sushi.SetorID, ModeloID: sushi.ID, Setor: sushi.SetorNome, TipoNegocio: sushi.TipoNome, Especialidade: sushi.Nome, PerecibilidadeSemanal: sushi.PerecibilidadeSemanal, ReputacaoImportancia: sushi.ReputacaoImportancia, DeliveryAfinidade: sushi.DeliveryAfinidade, CapitalProprio: 30000, Caixa: 10000, Preco: sushi.PrecoRef, PrecoReferencia: sushi.PrecoRef, CustoUnitario: sushi.CustoUnitario, AlcanceBase: sushi.AlcanceBase, ConversaoBase: sushi.ConversaoBase, RecorrenciaBase: sushi.RecorrenciaBase, CapacidadeBase: sushi.CapacidadeBase, SazonalidadeAmplitude: sushi.Sazonalidade, SazonalidadeFase: sushi.Fase, UsaEstoque: true, UsaInsumos: true, Insumos: ins, EstoqueValor: val, Operacao: "hibrida", Imovel: "alugado", AluguelMensal: sushi.Aluguel, QualidadeLocalizacao: "media", Delivery: true, DeliveryTipo: "misto", Funcionarios: sushi.Funcionarios, SalarioMedio: 1800, MarketingSemanal: 150, MeiosPagamento: []string{"pix"}, PrazoFornecedorMax: 2, Cenario: "Mercado estável", CenarioAlcance: 1, CenarioConversao: 1, CenarioOscilacao: 0, CenarioEventoNegExtra: 0, DuracaoSemanas: 12, ConcorrenciaNivel: "media", ConcorrenciaIndice: 1, Reputacao: 50, Canvas: LeanCanvas{Problema: "teste", Segmentos: []string{"qualidade"}, PropostaValor: "qualidade", Solucao: "sushi", Canais: []string{"redes_sociais"}, ReceitaModelo: "venda_unica"}, Hipoteses: Hipoteses{30, 2160, true, 15, 13}, InvestimentosIniciais: map[string]float64{"equipamentos": sushi.Equipamentos, "equipamentos_referencia": sushi.Equipamentos, "capital_giro_inicial": 10000}}
	f := sc.Fornecedores[1]
	ok2, c, _ := placeInputOrder(e, "salmao", 10, f, 1)
	if !ok2 || c <= 0 {
		return errors.New("pedido de insumo falhou")
	}
	if len(e.PedidosInsumos) == 0 {
		return errors.New("pedido não registrado")
	}
	r := processWeek(e)
	if r.CMV <= 0 {
		return errors.New("cmv de insumos inválido")
	}
	if r.DesperdicioInsumos < 0 {
		return errors.New("desperdício inválido")
	}
	if e.EstoqueValor < 0 {
		return errors.New("estoque negativo")
	}
	tmp, err := os.MkdirTemp("", "jedtest")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	p := filepath.Join(tmp, "e.json")
	if err := saveJSON(p, e); err != nil {
		return err
	}
	var e2 Empresa
	if err := loadJSON(p, &e2); err != nil {
		return err
	}
	if !e2.UsaInsumos || len(e2.Insumos) < 5 {
		return errors.New("persistência de insumos falhou")
	}
	// Migração simulada da v0.6: estoque genérico sem lista de insumos.
	oldCompany := *e
	oldCompany.UsaInsumos = false
	oldCompany.Insumos = nil
	oldCompany.PedidosInsumos = nil
	oldCompany.EstoqueValor = 4800
	oldCompany.EstoqueUnidades = 150
	oldCompany.VersaoDados = "0.6.0"
	pOld := filepath.Join(tmp, "old.json")
	if err := saveJSON(pOld, &oldCompany); err != nil {
		return err
	}
	migrated, err := loadCompany(pOld)
	if err != nil {
		return err
	}
	if !migrated.UsaInsumos || len(migrated.Insumos) < 5 {
		return errors.New("migração v0.6 falhou")
	}
	if normalizeDifficulty("x") != "intermediario" || difficultyProfile(&Empresa{Dificuldade: "iniciante"}).Oscilacao >= 1 {
		return errors.New("dificuldade falhou")
	}
	fmt.Printf("SELF-TEST OK | catálogo=%d | perfis=%d | insumos sushi=%d | CMV=%s | desperdício=%s | migração=v0.6→v2.0 OK | dificuldade=OK\n", catalogCount(cat), len(sc.Perfis), len(e.Insumos), money(r.CMV), money(r.DesperdicioInsumos))
	return nil
}

func main() {
	setUTF8Console()
	if serverMode == "true" {
		if err := runServerCLI(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "JED Servidor:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 {
		if os.Args[1] == "--server" {
			if err := runServerCLI(os.Args[2:]); err != nil {
				fmt.Fprintln(os.Stderr, "JED Servidor:", err)
				os.Exit(1)
			}
			return
		}
		switch os.Args[1] {
		case "--self-test":
			if err := selfTest(); err != nil {
				fmt.Fprintln(os.Stderr, "SELF-TEST FALHOU:", err)
				os.Exit(1)
			}
			return
		case "--version":
			fmt.Println("JED Simulador", version)
			return
		case "--classic":
			runClassicApp()
			return
		case "--balance-test":
			_ = ensureDirs()
			runs := 1
			diff := "intermediario"
			if len(os.Args) > 2 {
				if n, e := strconv.Atoi(os.Args[2]); e == nil && n > 0 {
					runs = n
				}
			}
			if len(os.Args) > 3 {
				diff = normalizeDifficulty(os.Args[3])
			}
			cat, er := loadCatalog()
			if er != nil {
				fmt.Fprintln(os.Stderr, er)
				os.Exit(1)
			}
			sc, er := loadSupplyCatalog()
			if er != nil {
				fmt.Fprintln(os.Stderr, er)
				os.Exit(1)
			}
			rows, p, er := runBalanceLab(cat, sc, runs, diff)
			if er != nil {
				fmt.Fprintln(os.Stderr, er)
				os.Exit(1)
			}
			fmt.Printf("BALANCE-TEST OK | runs=%d | dificuldade=%s | linhas=%d | %s\n", runs, diff, len(rows), p)
			return
		}
	}
	if runtime.GOOS == "windows" && forceClassic != "true" {
		if err := runNativeUI(); err == nil {
			return
		} else {
			fmt.Fprintln(os.Stderr, "Falha da GUI nativa:", err)
			showNativeError("JED Simulador — falha na interface", fmt.Sprintf("A interface nativa não conseguiu abrir.\n\nErro: %v\n\nO modo Classic será iniciado como alternativa.", err))
			runClassicApp()
			return
		}
	}
	runClassicApp()
}

func runClassicApp() {
	if err := ensureDirs(); err != nil {
		fmt.Println("Não foi possível criar as pastas de dados:", err)
		pause()
		return
	}
	cat, err := loadCatalog()
	if err != nil {
		fmt.Println("Erro ao carregar catálogo:", err)
		pause()
		return
	}
	var current *Empresa
	sc, err := loadSupplyCatalog()
	if err != nil {
		fmt.Println("Erro ao carregar catálogo de insumos:", err)
		pause()
		return
	}
	firstRun()
	for {
		header("PAINEL INICIAL  •  v" + version)
		fmt.Println("\n" + muted("Simulador educacional portátil • GUI nativa no Windows"))
		section("Começar")
		menuLine("1", "Novo negócio", "criar uma empresa do zero")
		menuLine("2", "Carregar empresa", "retomar uma simulação salva")
		if current != nil {
			menuLine("4", "Continuar", current.Nome+" • semana "+strconv.Itoa(current.Semana+1))
		}
		sectionEnd()
		section("Aprender e ensinar")
		menuLine("3", "Modo Tutor", "turmas, cenários, comparação e balanceamento")
		menuLine("5", "Tutorial", "aprender os conceitos essenciais")
		menuLine("6", "Aparência", "ligar ou desligar as cores")
		menuLine("H", "Ajuda", "guia rápido do simulador")
		menuLine("0", "Sair", "encerrar o programa")
		sectionEnd()
		fmt.Print("\nEscolha: ")
		x := strings.ToLower(readLine())
		switch x {
		case "1":
			e := createCompany(cat, sc)
			if e != nil {
				current = e
				_, _ = saveCompany(e)
				panel(e)
			}
		case "2":
			e := loadCompanyUI()
			if e != nil {
				current = e
				panel(e)
			}
		case "3":
			tutorMode()
		case "4":
			if current != nil {
				panel(current)
			}
		case "5":
			tutorialMode()
		case "6":
			appearanceUI()
		case "h":
			setHelp("geral")
			showHelp(helpTopic)
		case "0":
			return
		}
	}
}
