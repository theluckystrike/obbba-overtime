# obbba-overtime

[![Go Reference](https://pkg.go.dev/badge/github.com/theluckystrike/obbba-overtime.svg)](https://pkg.go.dev/github.com/theluckystrike/obbba-overtime)

The **One Big Beautiful Bill Act (OBBBA, 2025)** "no tax on overtime"
deduction — IRC section 225. The deduction is an **above-the-line** deduction
on the **FLSA overtime premium** (the half-time premium paid on hours worked
beyond 40 in a workweek), **not** on total overtime pay. Pure, dependency-free
Go math — the same engine behind the
[noovertimetax.com](https://noovertimetax.com/)
[overtime tax calculator](https://noovertimetax.com/overtime-tax-calculator/).

## Statutory parameters (2025-2028; sunsets after 2028)

| Parameter | Single | MFJ |
| --- | --- | --- |
| Premium fraction | 0.5 (FLSA time-and-a-half) | 0.5 |
| Per-return cap | $12,500 | $25,000 |
| MAGI phase-out starts | $150,000 | $300,000 |
| Fully phased out at | $275,000 | $550,000 |
| Phase-out slope | -$100 / +$1,000 MAGI | same |

FICA (7.65%) still applies to the full overtime pay — the deduction is
income-tax only.

## Install

```sh
go get github.com/theluckystrike/obbba-overtime
```

## Quick example

```go
package main

import (
	"fmt"

	obbaovertime "github.com/theluckystrike/obbba-overtime"
)

func main() {
	// $30/hr OT, 500 OT hours/yr, single, 22% bracket
	r := obbaovertime.Deduction(60_000.0, 30.0, 500.0, false, 0.22, false)
	fmt.Println(r.FLSAPremium) // 7500  (0.5 * 30 * 500)
	fmt.Println(r.Savings)     // 1650  (7500 * 0.22)

	// Explicit MAGI for the phase-out
	r2 := obbaovertime.DeductionWithMAGI(30.0, 1_500.0, false, 0.22, 200_000.0)
	fmt.Println(r2.Deductible) // 7500  (capped 12500 - 5000 phase-out)
}
```

## API

| Function | Description |
| --- | --- |
| `Deduction(annualWage, otRate, annualOtHours float64, joint bool, marginalRate float64, ficaOnOvertime bool) Result` | Full §225 deduction + federal income-tax savings. |
| `DeductionWithMAGI(otRate, annualOtHours float64, joint bool, marginalRate, magi float64) Result` | Same, with an explicit MAGI for the phase-out. |
| `FICA(grossOvertime float64) float64` | 7.65% FICA still owed on full overtime pay. |

## Verification

All figures verified against IRS / Joint Committee on Taxation / Tax
Foundation primary sources (2025-2028). Re-verify weekly; OBBBA provisions
sunset after 2028.

## License

MIT.

## Links

- Overtime tax calculator: <https://noovertimetax.com/overtime-tax-calculator/>
- Site: <https://noovertimetax.com/>
- Package docs: <https://pkg.go.dev/github.com/theluckystrike/obbba-overtime>
