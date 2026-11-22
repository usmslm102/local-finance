package retirement

import (
	"context"
	"math"
	"testing"
)

func fixture() Scenario {
	return Scenario{Age: 30, Horizon: 90, MonthlyExpense: 100000, RetirementExpense: 100000, ExpenseChangeYears: 2, Inflation: 6, WithdrawalRate: 3, RSUVestYears: 4, RSUNetPercent: 70, Assets: []Asset{{Kind: "MARKET", Name: "India", Currency: "INR", AccessiblePercent: 100}}}
}
func near(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > math.Max(0.01, math.Abs(want)*1e-8) {
		t.Fatalf("got %.8f want %.8f", got, want)
	}
}

func TestTargetAndImmediateRetirement(t *testing.T) {
	s := fixture()
	s.Assets[0].Balance = 50000000
	s.Assets[0].Return = 8
	result, err := Project(s)
	if err != nil {
		t.Fatal(err)
	}
	near(t, result.TodayTarget, 40000000)
	if result.RetirementAge == nil || *result.RetirementAge != 30 {
		t.Fatalf("expected immediate retirement %+v", result.RetirementAge)
	}
	if result.Points[len(result.Points)-1].Portfolio <= 0 {
		t.Fatal("horizon not funded")
	}
	for _, p := range result.Points {
		if p.MonthlyInvestment != 0 || p.Invested != 0 {
			t.Fatal("retirement kept contributing")
		}
	}
}

func TestTargetAloneDoesNotEstablishSustainability(t *testing.T) {
	s := fixture()
	s.Assets[0].Balance = 40000000
	s.Assets[0].Return = 0
	r, err := Project(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.RetirementAge != nil {
		t.Fatalf("fixed initial target incorrectly guaranteed 60 years: %v", *r.RetirementAge)
	}
	if len(r.Points) == 0 {
		t.Fatal("no fallback projection")
	}
	for _, p := range r.Points {
		if p.Retired {
			t.Fatal("infeasible scenario marked retired at horizon")
		}
	}
}

func TestProjectionNumericLimitsAndCancellation(t *testing.T) {
	s := fixture()
	s.WithdrawalRate = 1e-300
	if _, err := Project(s); err == nil {
		t.Fatal("infinite target accepted")
	}
	s = fixture()
	s.Age = 18
	s.Horizon = 105
	s.Assets[0].Balance = 1e12
	s.Assets[0].Return = 25
	if _, err := Project(s); err == nil {
		t.Fatal("numeric overflow returned success with null chart")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ProjectContext(ctx, fixture()); err != context.Canceled {
		t.Fatalf("canceled query: %v", err)
	}
}

func TestSurplusIsAllocatedWithoutFixedSIPs(t *testing.T) {
	s := fixture()
	s.Inflation = 0
	s.MonthlyIncome = 1000
	s.MonthlyExpense = 100
	s.RetirementExpense = 100
	s.InvestSurplus = true
	s.Assets[0].AccessAge = 60
	points, _, _ := simulate(s, 12, true)
	near(t, points[1].Invested, 10800)
	s.Assets[0].Kind = "PF"
	if _, err := Project(s); err == nil {
		t.Fatal("surplus with only payroll accounts accepted")
	}
}

func TestMonthEndFXContributionMatchesEquivalentINRReturn(t *testing.T) {
	s := fixture()
	s.Inflation = 0
	s.USDINR = 80
	s.FXGrowth = 12
	s.Assets[0].Currency = "USD"
	s.Assets[0].Monthly = 8000
	usd, _, _ := simulate(s, 720, true)
	s.USDINR = 0
	s.FXGrowth = 0
	s.Assets[0].Currency = "INR"
	s.Assets[0].Return = 12
	inr, _, _ := simulate(s, 720, true)
	for i := range usd {
		near(t, usd[i].Portfolio, inr[i].Portfolio)
		near(t, usd[i].Invested, inr[i].Invested)
	}
}

func TestContributionsStepUpAndStopAtRetirement(t *testing.T) {
	s := fixture()
	s.MonthlyExpense = 100
	s.RetirementExpense = 100
	s.Inflation = 0
	s.WithdrawalRate = 3
	s.ContributionGrowth = 10
	s.Assets[0].Monthly = 10000
	s.Assets[0].MonthlyIncrease = 100
	points, _, _ := simulate(s, 24, true)
	near(t, points[1].MonthlyInvestment, (10000+1200)*1.1)
	near(t, points[2].MonthlyInvestment, 0)
	for _, p := range points {
		if p.Age >= 32 && (p.MonthlyInvestment != 0 || p.Invested != points[2].Invested) {
			t.Fatal("contributions continued after retirement")
		}
	}
}

func TestLockedFundsCannotFundEarlyRetirement(t *testing.T) {
	s := fixture()
	s.Assets[0].Balance = 100000000
	s.Assets[0].AccessAge = 60
	s.Assets[0].Return = 8
	r, err := Project(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.RetirementAge == nil || *r.RetirementAge < 60 {
		t.Fatalf("locked wealth funded early retirement: %v", r.RetirementAge)
	}
	s.Assets[0].AccessiblePercent = 0
	r, err = Project(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.RetirementAge != nil {
		t.Fatal("unavailable funds were spent")
	}
}

func TestFXAndInflationAreAppliedOnce(t *testing.T) {
	s := fixture()
	s.USDINR = 80
	s.FXGrowth = 2
	s.Assets[0].Currency = "USD"
	s.Assets[0].Balance = 100
	s.Assets[0].Return = 5
	points, _, _ := simulate(s, 720, true)
	near(t, points[1].Portfolio, 100*80*1.05*1.02)
	near(t, points[1].TodayValue, 100*80*1.05*1.02/1.06)
	s.Assets[0].Balance = 0
	s.Assets[0].Return = 0
	s.Assets[0].Monthly = 8000
	s.FXGrowth = 0
	points, _, _ = simulate(s, 720, true)
	near(t, points[1].Invested, 96000)
	near(t, points[1].Portfolio, 96000)
}

func TestRSUCohortsTaxAndEmploymentCutoff(t *testing.T) {
	s := fixture()
	s.USDINR = 80
	s.Inflation = 0
	s.RSUAnnual = 12000
	s.RSUUnvested = 24000
	s.RSUVestYears = 2
	s.RSUNetPercent = 50
	s.Assets = []Asset{{Kind: "EMPLOYER", Name: "Work shares", Currency: "USD", AccessiblePercent: 100}}
	points, _, _ := simulate(s, 24, true)
	near(t, points[0].Portfolio, 0) // unvested is not starting capital
	near(t, points[1].Invested, (24000.0/24+12000.0/24)*12*0.5*80)
	near(t, points[2].Invested, ((24000.0/24+12000.0/24)*12+(24000.0/24+12000.0/24+12000.0/24)*12)*0.5*80)
	for _, p := range points {
		if p.Age >= 32 && p.Invested != points[2].Invested {
			t.Fatal("unvested grants kept vesting after employment stopped")
		}
	}
}

func TestSalarySurplusDoesNotReusePayrollDeductions(t *testing.T) {
	s := fixture()
	s.Inflation = 0
	s.MonthlyExpense = 100
	s.RetirementExpense = 100
	s.MonthlyIncome = 1000
	s.InvestSurplus = true
	s.Assets[0].Monthly = 200
	s.Assets = append(s.Assets, Asset{Kind: "PF", Name: "PF", Currency: "INR", Monthly: 100, AccessAge: 60, AccessiblePercent: 100})
	points, _, _ := simulate(s, 12, true)
	near(t, points[0].MonthlyInvestment, 1000) // 900 from net salary plus 100 already deducted for PF
	near(t, points[1].Invested, 12000)
}

func TestInvalidScenariosReturnErrors(t *testing.T) {
	for _, mutate := range []func(*Scenario){func(s *Scenario) { s.WithdrawalRate = 0 }, func(s *Scenario) { s.Age = 0 }, func(s *Scenario) { s.Assets[0].Return = math.NaN() }, func(s *Scenario) { s.Assets[0].Currency = "USD"; s.Assets[0].Balance = 1 }, func(s *Scenario) { s.Assets[0].Monthly = -1 }, func(s *Scenario) { s.Assets[0].Kind = "unknown" }} {
		s := fixture()
		mutate(&s)
		if _, err := Project(s); err == nil {
			t.Fatal("invalid inputs accepted")
		}
	}
}
