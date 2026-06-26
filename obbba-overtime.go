// Package obbaovertime implements the One Big Beautiful Bill Act (OBBBA, 2025)
// above-the-line deduction for "no tax on overtime" — IRC section 225 as
// enacted by P.L. 119-21. The deduction applies only to the FLSA overtime
// PREMIUM (the half-time premium paid on hours worked beyond 40 in a
// workweek), NOT to total overtime pay. It is the same math behind the
// noovertimetax.com overtime tax calculator.
//
// Statutory parameters (2025-2028; sunsets after 2028):
//
//   - Premium fraction: 0.5 (FLSA time-and-a-half premium on hours > 40).
//   - Cap: $12,500 single / $25,000 joint PER RETURN.
//   - MAGI phase-out: -$100 of deduction per $1,000 MAGI over
//     $150,000 single / $300,000 joint, fully reduced to $0 at
//     $275,000 single / $550,000 joint.
//   - FICA (7.65%) still applies to the full overtime pay — the deduction is
//     income-tax only.
//
// Quick example:
//
//	d := obbaovertime.Deduction(120_000.0, 30.0, 10.0, true, 0.22, false)
//	// d.Savings ~= the federal income-tax value of the deductible premium
package obbaovertime

// Filing status selects the cap and phase-out thresholds.
type FilingStatus int

const (
	Single FilingStatus = iota
	Joint
)

const (
	flsaPremiumFraction = 0.5

	capSingle = 12_500.0
	capJoint  = 25_000.0

	phaseoutStartSingle = 150_000.0
	phaseoutStartJoint  = 300_000.0

	// Phase-out fully zeroes the deduction at these MAGI values.
	phaseoutEndSingle = 275_000.0
	phaseoutEndJoint  = 550_000.0
)

// Result holds the components of the OBBBA overtime deduction.
type Result struct {
	GrossOvertime  float64 // total overtime pay (hours x rate)
	FLSAPremium    float64 // deductible half-time premium (the §225 base)
	CappedPremium  float64 // premium after the $12.5k/$25k per-return cap
	Deductible     float64 // premium after MAGI phase-out (the actual §225 deduction)
	MarginalRate   float64 // marginal rate applied to the deductible amount
	Savings        float64 // federal income-tax savings = Deductible x MarginalRate
	CapApplied     bool    // true if the per-return cap bound the deduction
	PhasedOut      bool    // true if MAGI phase-out reduced the deduction
	FullyPhasedOut bool    // true if MAGI reduced the deduction to $0
}

// premium returns the FLSA overtime premium (half-time premium on hours
// beyond 40) for a given pay period.
func premium(rate, otHours float64) float64 {
	if otHours <= 0.0 || rate <= 0.0 {
		return 0.0
	}
	return flsaPremiumFraction * rate * otHours
}

// cap returns the per-return cap for a filing status.
func capFor(fs FilingStatus) float64 {
	if fs == Joint {
		return capJoint
	}
	return capSingle
}

// phaseOut reduces the (already capped) deduction for MAGI above the
// threshold: -$100 per $1,000 MAGI over the start, floored at 0. This matches
// the statutory linear reduction that hits $0 at phaseoutEndSingle/Joint.
func phaseOut(capped, magi float64, fs FilingStatus) (after float64, phasedOut, fullyGone bool) {
	start := phaseoutStartSingle
	end := phaseoutEndSingle
	if fs == Joint {
		start = phaseoutStartJoint
		end = phaseoutEndJoint
	}
	if magi <= start {
		return capped, false, false
	}
	if magi >= end {
		return 0.0, true, true
	}
	// -$100 per full $1,000 over the start.
	reduction := 100.0 * float64(int((magi-start)/1_000.0))
	after = capped - reduction
	if after < 0.0 {
		after = 0.0
	}
	return after, true, false
}

// Deduction computes the OBBBA §225 overtime deduction and its federal
// income-tax savings for one return.
//
//   - annualWage:      regular annual wages (not used by the deduction, but
//                      typically needed to identify the marginal bracket).
//   - otRate:          hourly overtime pay rate (base rate, not premium).
//   - annualOtHours:   overtime hours worked over the year beyond 40/week.
//   - joint:           filing status (true = MFJ).
//   - marginalRate:    federal marginal income-tax rate on the last dollar
//                      (the deduction is above-the-line, valued at this rate).
//   - ficaOnOvertime:  if true, Savings is reduced by the 7.65% FICA that still
//                      applies to the full overtime pay (off by default — the
//                      headline "no tax on overtime" figure is income-tax only).
func Deduction(annualWage, otRate, annualOtHours float64, joint bool, marginalRate float64, ficaOnOvertime bool) Result {
	_ = annualWage // reserved; marginal bracket selection happens upstream
	fs := Single
	if joint {
		fs = Joint
	}

	gross := otRate * annualOtHours
	prem := premium(otRate, annualOtHours)

	capped := prem
	capHit := false
	if capped > capFor(fs) {
		capped = capFor(fs)
		capHit = true
	}

	// magi defaults to annualWage + premium when no separate MAGI is supplied;
	// callers needing a precise MAGI should use DeductionWithMAGI.
	deductible, phasedOut, fullyGone := phaseOut(capped, annualWage+prem, fs)

	savings := deductible * marginalRate
	if ficaOnOvertime {
		savings -= gross * 0.0765
		if savings < 0.0 {
			savings = 0.0
		}
	}

	return Result{
		GrossOvertime:  gross,
		FLSAPremium:    prem,
		CappedPremium:  capped,
		Deductible:     deductible,
		MarginalRate:   marginalRate,
		Savings:        savings,
		CapApplied:     capHit,
		PhasedOut:      phasedOut,
		FullyPhasedOut: fullyGone,
	}
}

// DeductionWithMAGI is like Deduction but takes an explicit MAGI (modified
// adjusted gross income) for the phase-out calculation, which is how the
// statute actually phases the deduction out.
func DeductionWithMAGI(otRate, annualOtHours float64, joint bool, marginalRate, magi float64) Result {
	fs := Single
	if joint {
		fs = Joint
	}

	prem := premium(otRate, annualOtHours)
	capped := prem
	if capped > capFor(fs) {
		capped = capFor(fs)
	}
	deductible, phasedOut, fullyGone := phaseOut(capped, magi, fs)

	return Result{
		GrossOvertime:  otRate * annualOtHours,
		FLSAPremium:    prem,
		CappedPremium:  capped,
		Deductible:     deductible,
		MarginalRate:   marginalRate,
		Savings:        deductible * marginalRate,
		PhasedOut:      phasedOut,
		FullyPhasedOut: fullyGone,
	}
}

// FICA returns the combined 7.65% FICA (6.2% Social Security + 1.45%
// Medicare) owed on the full overtime pay. The §225 deduction does NOT remove
// this — FICA still applies to total overtime pay.
func FICA(grossOvertime float64) float64 {
	if grossOvertime <= 0.0 {
		return 0.0
	}
	return grossOvertime * 0.0765
}
