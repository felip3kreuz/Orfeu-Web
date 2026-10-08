package core

import "math/rand"

// Simulator is the small stateful API intended for non-native frontends.
// It owns only the pseudo-random source. All business state continues to live
// in Empresa, making the same Core usable by Win32, WebAssembly and tests.
type Simulator struct {
	rng *rand.Rand
}

// NewSimulator creates a deterministic simulator. Production frontends should
// choose their seed explicitly; tests can replay a run by reusing the seed.
func NewSimulator(seed int64) *Simulator {
	return &Simulator{rng: rand.New(rand.NewSource(seed))}
}

// ProcessWeek advances one company by one simulated week.
func (s *Simulator) ProcessWeek(e *Empresa) Registro {
	return ProcessWeek(e, s.rng)
}

// RandomSource exposes the simulator-owned deterministic RNG to bridge helpers
// that must follow the same random stream as weekly processing.
func (s *Simulator) RandomSource() RandomSource {
	return s.rng
}

// Indicators returns the current aggregate indicators.
func (s *Simulator) Indicators(e *Empresa) *Indicadores {
	return Indicators(e)
}

// Score returns the current canonical JED score.
func (s *Simulator) Score(e *Empresa) Score {
	return CalculateScore(e)
}

// Review returns the hypothesis review when the current week closes a
// four-week review cycle; otherwise it returns nil.
func (s *Simulator) Review(e *Empresa) *Review {
	return ReviewHypotheses(e)
}

// DigitalChannels returns a copy of the canonical channel catalog.
func (s *Simulator) DigitalChannels() []CanalDigitalSpec {
	return DigitalChannelSpecs()
}

// DigitalTools returns a copy of the canonical digital-tool catalog.
func (s *Simulator) DigitalTools() []FerramentaDigitalSpec {
	return DigitalToolSpecs()
}
