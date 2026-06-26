package obbaovertime

import (
	"math"
	"testing"
)

func approx(a, b, tol float64) bool { return math.Abs(a-b) < tol }

// Worker $60k base, $30/hr OT rate, 500 OT hours/yr, single, 22% bracket.
// Premium = 0.5 * 30 * 500 = 7500. Well under cap, no phase-out.
// Savings = 7500 * 0.22 = 1650.
func TestTypicalWorker(t *testing.T) {
	r := Deduction(60_000.0, 30.0, 500.0, false, 0.22, false)
	if !approx(r.FLSAPremium, 7_500.0, 1e-9) {
		t.Fatalf("premium = %v, want 7500", r.FLSAPremium)
	}
	if !approx(r.Deductible, 7_500.0, 1e-9) {
		t.Fatalf("deductible = %v, want 7500 (no cap, no phase-out)", r.Deductible)
	}
	if !approx(r.Savings, 1_650.0, 1e-9) {
		t.Fatalf("savings = %v, want 1650", r.Savings)
	}
	if r.CapApplied || r.PhasedOut || r.FullyPhasedOut {
		t.Fatal("flags should all be false for a typical worker")
	}
}

// High-OT worker: $30/hr, 1500 OT hours -> premium 0.5*30*1500 = 22500.
// Single cap = 12500, so capped to 12500.
func TestCapSingle(t *testing.T) {
	r := Deduction(60_000.0, 30.0, 1_500.0, false, 0.22, false)
	if !approx(r.FLSAPremium, 22_500.0, 1e-9) {
		t.Fatalf("premium = %v, want 22500", r.FLSAPremium)
	}
	if !approx(r.CappedPremium, 12_500.0, 1e-9) {
		t.Fatalf("capped = %v, want 12500 (single cap)", r.CappedPremium)
	}
	if !r.CapApplied {
		t.Fatal("CapApplied should be true")
	}
}

// Joint return: $30/hr, 1500 OT hours -> premium 22500, under joint cap 25000.
func TestJointCapNotHit(t *testing.T) {
	r := Deduction(200_000.0, 30.0, 1_500.0, true, 0.24, false)
	if !approx(r.CappedPremium, 22_500.0, 1e-9) {
		t.Fatalf("capped = %v, want 22500 (under joint 25k cap)", r.CappedPremium)
	}
	if r.CapApplied {
		t.Fatal("CapApplied should be false under joint cap")
	}
}

// Phase-out: single MAGI = $200k. Reduction = $100 * int((200000-150000)/1000)
// = $100 * 50 = $5000. Capped premium 12500 -> deductible 7500.
func TestPhaseoutPartial(t *testing.T) {
	r := DeductionWithMAGI(30.0, 1_500.0, false, 0.22, 200_000.0)
	if !approx(r.CappedPremium, 12_500.0, 1e-9) {
		t.Fatalf("capped = %v, want 12500", r.CappedPremium)
	}
	if !approx(r.Deductible, 7_500.0, 1e-9) {
		t.Fatalf("deductible = %v, want 7500 after -$5000 phase-out", r.Deductible)
	}
	if !r.PhasedOut || r.FullyPhasedOut {
		t.Fatal("PhasedOut=true, FullyPhasedOut=false expected")
	}
}

// Fully phased out at the end threshold: single MAGI = $275k -> $0.
func TestPhaseoutFullSingle(t *testing.T) {
	r := DeductionWithMAGI(30.0, 1_500.0, false, 0.22, 275_000.0)
	if r.Deductible != 0.0 {
		t.Fatalf("deductible = %v, want 0 (fully phased out at $275k single)", r.Deductible)
	}
	if !r.FullyPhasedOut {
		t.Fatal("FullyPhasedOut should be true at $275k single")
	}
}

// Joint full phase-out at $550k.
func TestPhaseoutFullJoint(t *testing.T) {
	r := DeductionWithMAGI(30.0, 1_500.0, true, 0.24, 550_000.0)
	if r.Deductible != 0.0 {
		t.Fatalf("deductible = %v, want 0 (fully phased out at $550k joint)", r.Deductible)
	}
}

// FICA still applies to the full OT pay even though the premium is deductible.
func TestFICA(t *testing.T) {
	// 30/hr * 500h = 15000 gross OT -> FICA 7.65% = 1147.5
	if !approx(FICA(15_000.0), 1_147.5, 1e-9) {
		t.Fatalf("FICA = %v, want 1147.5", FICA(15_000.0))
	}
}

// Premium fraction is the FLSA half-time premium, not total OT pay.
func TestPremiumIsHalfTime(t *testing.T) {
	// 10 hrs at $40/hr -> base OT pay = rate*hours = $400; the deductible
	// premium (the half-time portion) = 0.5*40*10 = $200.
	r := Deduction(0.0, 40.0, 10.0, false, 0.0, false)
	if !approx(r.GrossOvertime, 400.0, 1e-9) {
		t.Fatalf("gross = %v, want 400 (base OT pay = rate x hours)", r.GrossOvertime)
	}
	if !approx(r.FLSAPremium, 200.0, 1e-9) {
		t.Fatalf("premium = %v, want 200 (half-time premium)", r.FLSAPremium)
	}
}

func TestNegativeHoursGuarded(t *testing.T) {
	r := Deduction(50_000.0, 30.0, -10.0, false, 0.22, false)
	if r.FLSAPremium != 0.0 || r.Savings != 0.0 {
		t.Fatal("negative hours should yield zero premium/savings")
	}
}
