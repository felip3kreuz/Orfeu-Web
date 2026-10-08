package main

// W1.2 engine compatibility facade.
//
// Deterministic simulation calculations now live in internal/core. These
// aliases and wrappers preserve the RC1.8 package-main API while Win32,
// Classic UI, server and remaining engine code are extracted incrementally.

import core "jed-simulador/internal/core"

type Mods = core.Mods
type DifficultyProfile = core.DifficultyProfile

func contains(xs []string, s string) bool                      { return core.Contains(xs, s) }
func canvasMods(e *Empresa) Mods                               { return core.CanvasMods(e) }
func canvasExplanations(e *Empresa) []string                   { return core.CanvasExplanations(e) }
func clamp(v, lo, hi float64) float64                          { return core.Clamp(v, lo, hi) }
func round2(v float64) float64                                 { return core.Round2(v) }
func locationFactor(e *Empresa) float64                        { return core.LocationFactor(e) }
func seasonFactor(e *Empresa, week int) float64                { return core.SeasonFactor(e, week) }
func competitionFactor(e *Empresa) float64                     { return core.CompetitionFactor(e) }
func equipmentFactor(e *Empresa) float64                       { return core.EquipmentFactor(e) }
func weeklyInterest(e *Empresa) float64                        { return core.WeeklyInterest(e) }
func priceFactor(e *Empresa, m Mods, p float64) float64        { return core.PriceFactor(e, m, p) }
func marketingFactor(e *Empresa, m Mods) float64               { return core.MarketingFactor(e, m) }
func launchFactor(e *Empresa) float64                          { return core.LaunchFactor(e) }
func repReach(e *Empresa) float64                              { return core.RepReach(e) }
func repConv(e *Empresa) float64                               { return core.RepConv(e) }
func repRec(e *Empresa) float64                                { return core.RepRec(e) }
func normalizeDifficulty(d string) string                      { return core.NormalizeDifficulty(d) }
func difficultyProfile(e *Empresa) DifficultyProfile           { return core.DifficultyProfileFor(e) }
func capacity(e *Empresa, m Mods) int                          { return core.Capacity(e, m) }
func paymentWeights(e *Empresa) map[string]float64             { return core.PaymentWeights(e) }
func liquidate(list []Conta, week int) (float64, []Conta)      { return core.Liquidate(list, week) }
func sumAccounts(xs []Conta) float64                           { return core.SumAccounts(xs) }
func maxInt(a, b int) int                                      { return core.MaxInt(a, b) }
func minInt(a, b int) int                                      { return core.MinInt(a, b) }
func stockValueInputs(e *Empresa) float64                      { return core.StockValueInputs(e) }
func findInputIndex(e *Empresa, id string) int                 { return core.FindInputIndex(e, id) }
func receiveInputOrders(e *Empresa) string                     { return core.ReceiveInputOrders(e) }
func maxSalesByInputs(e *Empresa) (int, string)                { return core.MaxSalesByInputs(e) }
func consumeInputs(e *Empresa, sales int) float64              { return core.ConsumeInputs(e, sales) }
func buyStock(e *Empresa, q, term int) (bool, float64, string) { return core.BuyStock(e, q, term) }
func allocateSales(sales, newWant, recWant int) (int, int) {
	return core.AllocateSales(sales, newWant, recWant)
}
func personaCompleteness(p Persona) float64 { return core.PersonaCompleteness(p) }
func journeyStep(e *Empresa) int            { return core.JourneyStep(e) }

// W1.3 random-dependent engine compatibility facade. The legacy package keeps
// owning the process-wide RNG for now, while the Core receives it explicitly.
func randRange(a, b float64) float64              { return core.RandRange(rng, a, b) }
func updateCompetition(e *Empresa, delta float64) { core.UpdateCompetition(e, rng, delta) }
func selectEvent(e *Empresa) *Evento              { return core.SelectEvent(e, rng) }
func wasteInputs(e *Empresa) (float64, string)    { return core.WasteInputs(e, rng) }
func placeInputOrder(e *Empresa, inputID string, q float64, f FornecedorSpec, term int) (bool, float64, string) {
	return core.PlaceInputOrder(e, inputID, q, f, term, rng)
}

// W1.4 weekly-engine compatibility facade. Digital simulation parameters and
// the full weekly transition now live in Core; package main keeps the legacy
// names expected by Win32 and Classic UI while migration continues.
var canaisDigitais = core.DigitalChannelSpecs()
var ferramentasDigitais = core.DigitalToolSpecs()

func digitalToolSpec(id string) (FerramentaDigitalSpec, bool) { return core.DigitalToolSpec(id) }
func hasDigitalTool(e *Empresa, id string) bool               { return core.HasDigitalTool(e, id) }
func digitalToolCost(e *Empresa) float64                      { return core.DigitalToolCost(e) }
func digitalToolMods(e *Empresa) (reach, engagement, conv, rec float64) {
	return core.DigitalToolMods(e)
}
func digitalChannelMods(e *Empresa) (reach, conv, rec float64) { return core.DigitalChannelMods(e) }
func hasDigitalChannel(e *Empresa, id string) bool             { return core.HasDigitalChannel(e, id) }
func channelName(id string) string                             { return core.ChannelName(id) }
func fixedWeekly(e *Empresa) float64                           { return core.FixedWeekly(e) }
func processWeek(e *Empresa) Registro                          { return core.ProcessWeek(e, rng) }

// W1.5 web-facing Core facade compatibility. Catalog traversal and aggregate
// analytics no longer depend on package main; the legacy UI keeps the old
// lowercase names while WebAssembly can call the exported Core API directly.
func supplyProfileFor(m Modelo, c CatalogoInsumos) PerfilInsumos { return core.SupplyProfileFor(m, c) }
func findSupplier(c CatalogoInsumos, id string) (FornecedorSpec, bool) {
	return core.FindSupplier(c, id)
}
func catalogCount(c Catalogo) int                    { return core.CatalogCount(c) }
func findModel(c Catalogo, id string) (Modelo, bool) { return core.FindModel(c, id) }
func indicators(e *Empresa) *Indicadores             { return core.Indicators(e) }
func score(e *Empresa) Score                         { return core.CalculateScore(e) }
func reviewHypotheses(e *Empresa) *Review            { return core.ReviewHypotheses(e) }
