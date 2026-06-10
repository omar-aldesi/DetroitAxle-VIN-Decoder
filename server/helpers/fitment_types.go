package helpers

type FitResult int

const (
	FitNone     FitResult = 0 // required condition failed or callout explicitly mismatched
	FitWithNote FitResult = 1 // required conditions pass, but a callout field is missing
	FitExact    FitResult = 2 // all required conditions + all callouts verified
)

type EvalResult struct {
	Result   FitResult
	Notes    []string // callout notes — populated for FitWithNote
	RuleNote string   // the matched rule's own descriptive note (e.g. "14.29 inch rotor")
}
