// Package retirement calculates explicit, deterministic planning scenarios.
// Returns and access assumptions are user inputs, never forecasts or tax rules.
package retirement

import (
	"context"
	"errors"
	"math"
)

type Asset struct {
	Kind              string  `json:"kind"`
	Name              string  `json:"name"`
	Currency          string  `json:"currency"`
	Balance           float64 `json:"balance"`
	Monthly           float64 `json:"monthly"`
	MonthlyIncrease   float64 `json:"monthly_increase"`
	Return            float64 `json:"return_percent"`
	AccessAge         float64 `json:"access_age"`
	AccessiblePercent float64 `json:"accessible_percent"`
}
type Scenario struct {
	Age                int     `json:"age"`
	Horizon            int     `json:"horizon"`
	MonthlyExpense     float64 `json:"monthly_expense"`
	RetirementExpense  float64 `json:"retirement_expense"`
	ExpenseChangeYears int     `json:"expense_change_years"`
	Inflation          float64 `json:"inflation_percent"`
	WithdrawalRate     float64 `json:"withdrawal_percent"`
	ContributionGrowth float64 `json:"contribution_growth_percent"`
	MonthlyIncome      float64 `json:"monthly_income"`
	IncomeGrowth       float64 `json:"income_growth_percent"`
	InvestSurplus      bool    `json:"invest_surplus"`
	USDINR             float64 `json:"usd_inr"`
	FXGrowth           float64 `json:"fx_growth_percent"`
	RSUUnvested        float64 `json:"rsu_unvested_usd"`
	RSUAnnual          float64 `json:"rsu_annual_usd"`
	RSUVestYears       int     `json:"rsu_vest_years"`
	RSUGrantGrowth     float64 `json:"rsu_grant_growth_percent"`
	RSUNetPercent      float64 `json:"rsu_net_percent"`
	Assets             []Asset `json:"assets"`
}
type Point struct {
	Age               float64 `json:"age"`
	Portfolio         float64 `json:"portfolio"`
	Accessible        float64 `json:"accessible"`
	TodayValue        float64 `json:"today_value"`
	Target            float64 `json:"target"`
	Invested          float64 `json:"invested"`
	MonthlyInvestment float64 `json:"monthly_investment"`
	MonthlyExpense    float64 `json:"monthly_expense"`
	Retired           bool    `json:"retired"`
}
type Result struct {
	RetirementAge *float64 `json:"retirement_age"`
	TodayTarget   float64  `json:"today_target"`
	AtRetirement  *Point   `json:"at_retirement"`
	Points        []Point  `json:"points"`
	Warnings      []string `json:"warnings"`
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func (s Scenario) Validate() error {
	if s.Age < 18 || s.Age > 80 || s.Horizon <= s.Age || s.Horizon > 105 {
		return errors.New("enter age 18–80 and a planning horizon after your age, up to 105")
	}
	if s.MonthlyExpense <= 0 || s.RetirementExpense <= 0 || s.ExpenseChangeYears < 0 || s.ExpenseChangeYears > 50 {
		return errors.New("enter positive current and future monthly spending, and valid years until spending changes")
	}
	if s.WithdrawalRate <= 0 || s.WithdrawalRate > 10 || s.Inflation < 0 || s.Inflation > 25 {
		return errors.New("withdrawal rate must be above 0 and at most 10%; inflation must be 0–25%")
	}
	target := s.RetirementExpense * 12 / (s.WithdrawalRate / 100) * math.Pow(1+s.Inflation/100, float64(s.Horizon-s.Age))
	if !finite(target) || target > 1e18 {
		return errors.New("withdrawal target exceeds supported numeric limits; check spending, inflation and withdrawal rate")
	}
	if s.ContributionGrowth < 0 || s.ContributionGrowth > 30 || s.IncomeGrowth < 0 || s.IncomeGrowth > 30 || s.FXGrowth < -10 || s.FXGrowth > 15 || s.RSUGrantGrowth < 0 || s.RSUGrantGrowth > 30 {
		return errors.New("growth assumptions are outside supported planning bounds")
	}
	if len(s.Assets) < 1 || len(s.Assets) > 12 || s.RSUVestYears < 1 || s.RSUVestYears > 10 || s.RSUNetPercent < 0 || s.RSUNetPercent > 100 {
		return errors.New("provide 1–12 asset buckets, vesting years 1–10 and RSU after-tax retention 0–100%")
	}
	nums := []float64{s.MonthlyExpense, s.RetirementExpense, s.Inflation, s.WithdrawalRate, s.ContributionGrowth, s.MonthlyIncome, s.IncomeGrowth, s.USDINR, s.FXGrowth, s.RSUUnvested, s.RSUAnnual, s.RSUGrantGrowth, s.RSUNetPercent}
	usd := s.RSUUnvested > 0 || s.RSUAnnual > 0
	employer := false
	surplusBucket := false
	fallbackCurrency := ""
	employerCount := 0
	for _, a := range s.Assets {
		if a.Kind != "PF" && a.Kind != "NPS" && a.Kind != "EMPLOYER" {
			surplusBucket = true
			if fallbackCurrency == "" {
				fallbackCurrency = a.Currency
			}
		}
		if a.Kind != "MARKET" && a.Kind != "EMPLOYER" && a.Kind != "PF" && a.Kind != "NPS" && a.Kind != "OTHER" {
			return errors.New("choose a supported asset kind")
		}
		if len(a.Name) == 0 || len(a.Name) > 100 || a.Balance < 0 || a.Monthly < 0 || a.MonthlyIncrease < 0 || a.Return < -20 || a.Return > 25 || a.AccessAge < 0 || a.AccessAge > 105 || a.AccessiblePercent < 0 || a.AccessiblePercent > 100 || (a.Currency != "INR" && a.Currency != "USD") {
			return errors.New("check asset names, currency, nonnegative amounts, returns (-20–25%), access age and accessible percentage")
		}
		nums = append(nums, a.Balance, a.Monthly, a.MonthlyIncrease, a.Return, a.AccessAge, a.AccessiblePercent)
		if a.Currency == "USD" && (a.Balance > 0 || a.Monthly > 0 || a.MonthlyIncrease > 0) {
			usd = true
		}
		if a.Kind == "EMPLOYER" && a.Currency == "USD" {
			employer = true
			employerCount++
		}
	}
	if employerCount > 1 {
		return errors.New("use one USD employer-stock bucket for RSUs")
	}
	if s.InvestSurplus && !surplusBucket {
		return errors.New("surplus investing requires a market/deposit asset in addition to payroll accounts")
	}
	for _, v := range nums {
		if !finite(v) || math.Abs(v) > 1e12 {
			return errors.New("all inputs must be finite and within supported numeric limits")
		}
	}
	if s.InvestSurplus && fallbackCurrency == "USD" {
		usd = true
	}
	if s.MonthlyIncome < 0 || s.RSUUnvested < 0 || s.RSUAnnual < 0 || s.USDINR < 0 {
		return errors.New("income, grants and exchange rate cannot be negative")
	}
	if usd && (s.USDINR <= 0 || s.USDINR > 1000) {
		return errors.New("enter the INR per USD conversion rate for USD assets and RSUs")
	}
	if (s.RSUUnvested > 0 || s.RSUAnnual > 0) && !employer {
		return errors.New("RSUs require a USD employer-stock asset")
	}
	return nil
}

type balance struct {
	liquid, locked float64
	released       bool
}
type grant struct {
	monthly   float64
	remaining int
}

func Project(s Scenario) (Result, error) { return ProjectContext(context.Background(), s) }

func ProjectContext(ctx context.Context, s Scenario) (Result, error) {
	if err := s.Validate(); err != nil {
		return Result{}, err
	}
	result := Result{TodayTarget: s.RetirementExpense * 12 / (s.WithdrawalRate / 100), Warnings: []string{"Deterministic assumptions, not a probability of success. Market sequence risk, healthcare shocks and future tax/withdrawal law changes are not forecast.", "Returns should be entered after expected fees/taxes. Broker funding is a contribution estimate, not proof of holdings or investment performance.", "All contributions, ESPP, PF/NPS payments and employment-dependent RSU vesting/grants stop at retirement. Unvested stock is not starting wealth.", "PF/NPS access ages and accessible percentages are planning inputs, not statements of legal withdrawal eligibility. Unavailable portions never fund spending.", "INR spending inflates at the selected Indian inflation rate. USD returns and INR depreciation combine multiplicatively; US inflation is not subtracted a second time."}}
	months := (s.Horizon - s.Age) * 12
	var selected = -1
	// Each candidate must cover its target from accessible assets AND fund every
	// subsequent monthly withdrawal to the horizon, without any new contributions.
	for candidate := 0; candidate < months; candidate++ {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		_, at, ok := simulateContext(ctx, s, candidate, false)
		if ok && at.Accessible >= at.Target {
			selected = candidate
			break
		}
	}
	if s.MonthlyIncome > 0 {
		workingMonths := months
		if selected >= 0 {
			workingMonths = selected
		}
		for m := 0; m < workingMonths; m++ {
			year := m / 12
			cost := s.MonthlyExpense
			if m >= s.ExpenseChangeYears*12 {
				cost = s.RetirementExpense
			}
			cost *= math.Pow(1+s.Inflation/100, float64(m)/12)
			fixed := 0.0
			for _, a := range s.Assets {
				if a.Kind != "PF" && a.Kind != "NPS" && a.Kind != "EMPLOYER" {
					fixed += (a.Monthly + a.MonthlyIncrease*float64(m)) * math.Pow(1+s.ContributionGrowth/100, float64(year))
				}
			}
			if cost+fixed > s.MonthlyIncome*math.Pow(1+s.IncomeGrowth/100, float64(year))+0.01 {
				result.Warnings = append(result.Warnings, "Planned bank-funded contributions and living costs exceed projected net salary in at least one working month. The model retains your explicit contributions; confirm other funding or reduce them.")
				break
			}
		}
	}
	if selected < 0 {
		result.Points, _, _ = simulateContext(ctx, s, months+1, true)
		if ctx.Err() != nil {
			return Result{}, ctx.Err()
		}
		if len(result.Points) == 0 {
			return Result{}, errors.New("projection exceeds supported numeric limits; lower balances or growth assumptions")
		}
		result.Warnings = append(result.Warnings, "No retirement month satisfies both the withdrawal target and the horizon funding check under these assumptions.")
		return result, nil
	}
	points, at, _ := simulateContext(ctx, s, selected, true)
	if ctx.Err() != nil {
		return Result{}, ctx.Err()
	}
	if len(points) == 0 {
		return Result{}, errors.New("projection exceeds supported numeric limits")
	}
	result.Points, result.AtRetirement = points, &at
	age := float64(s.Age) + float64(selected)/12
	result.RetirementAge = &age
	return result, nil
}

func simulate(s Scenario, retire int, collect bool) ([]Point, Point, bool) {
	return simulateContext(context.Background(), s, retire, collect)
}
func simulateContext(ctx context.Context, s Scenario, retire int, collect bool) ([]Point, Point, bool) {
	months := (s.Horizon - s.Age) * 12
	balances := make([]balance, len(s.Assets))
	for i, a := range s.Assets {
		balances[i].locked = a.Balance
	}
	grants := []grant{}
	if s.RSUUnvested > 0 {
		grants = append(grants, grant{s.RSUUnvested / float64(s.RSUVestYears*12), s.RSUVestYears * 12})
	}
	employer := 0
	for i, a := range s.Assets {
		if a.Kind == "EMPLOYER" && a.Currency == "USD" {
			employer = i
		}
	}
	points := []Point{}
	at := Point{}
	invested := 0.0
	ok := true
	rate := func(a Asset, m int) float64 {
		if a.Currency == "USD" {
			if s.USDINR == 0 {
				return 1
			}
			return s.USDINR * math.Pow(1+s.FXGrowth/100, float64(m)/12)
		}
		return 1
	}
	expense := func(m int, retired bool) float64 {
		base := s.MonthlyExpense
		if retired || m >= s.ExpenseChangeYears*12 {
			base = s.RetirementExpense
		}
		return base * math.Pow(1+s.Inflation/100, float64(m)/12)
	}
	for m := 0; m <= months; m++ {
		if ctx.Err() != nil {
			return nil, at, false
		}
		age := float64(s.Age) + float64(m)/12
		for i, a := range s.Assets {
			if !balances[i].released && age >= a.AccessAge {
				move := balances[i].locked * a.AccessiblePercent / 100
				balances[i].liquid += move
				balances[i].locked -= move
				balances[i].released = true
			}
		}
		total, liquid := 0.0, 0.0
		for i, a := range s.Assets {
			fx := rate(a, m)
			total += (balances[i].liquid + balances[i].locked) * fx
			liquid += balances[i].liquid * fx
		}
		retired := m >= retire
		spending := expense(m, retired)
		point := Point{Age: age, Portfolio: total, Accessible: liquid, TodayValue: total / math.Pow(1+s.Inflation/100, float64(m)/12), Target: s.RetirementExpense * math.Pow(1+s.Inflation/100, float64(m)/12) * 12 / (s.WithdrawalRate / 100), Invested: invested, MonthlyExpense: spending, Retired: retired}
		if m == retire {
			at = point
			if !collect && point.Accessible < point.Target {
				return nil, at, false
			}
		}
		if collect && (m%12 == 0 || m == retire || m == months) {
			points = append(points, point)
		}
		if !finite(total) || !finite(point.Target) || total > 1e18 {
			return nil, at, false
		}
		if m == months {
			break
		}
		if retired {
			// Withdraw proportionally from accessible assets in INR. Locked funds cannot
			// bridge early retirement, and later release uses the user's explicit inputs.
			if liquid+0.0001 < spending {
				ok = false
				if !collect {
					return nil, at, false
				}
			}
			fraction := 0.0
			if liquid > 0 {
				fraction = math.Min(1, spending/liquid)
			}
			for i := range balances {
				balances[i].liquid *= 1 - fraction
			}
		}
		for i, a := range s.Assets {
			growth := math.Pow(1+a.Return/100, 1.0/12)
			balances[i].liquid *= growth
			balances[i].locked *= growth
		}
		if retired {
			continue
		}
		year := m / 12
		contributions := make([]float64, len(s.Assets))
		bankFunded := 0.0
		for i, a := range s.Assets {
			contributions[i] = (a.Monthly + a.MonthlyIncrease*float64(m)) * math.Pow(1+s.ContributionGrowth/100, float64(year)) / rate(a, m+1)
			if a.Kind != "PF" && a.Kind != "NPS" && a.Kind != "EMPLOYER" {
				bankFunded += contributions[i] * rate(a, m+1)
			}
		}
		if s.InvestSurplus {
			surplus := math.Max(0, s.MonthlyIncome*math.Pow(1+s.IncomeGrowth/100, float64(year))-expense(m, false)-bankFunded)
			// Use SIP weights, then current market/deposit values, then the first
			// market/deposit bucket. Never reuse payroll deductions or drop surplus.
			if bankFunded > 0 {
				for i, a := range s.Assets {
					if a.Kind != "PF" && a.Kind != "NPS" && a.Kind != "EMPLOYER" {
						contributions[i] += surplus * (contributions[i] * rate(a, m+1) / bankFunded) / rate(a, m+1)
					}
				}
			} else {
				weights := make([]float64, len(s.Assets))
				weightTotal := 0.0
				fallback := -1
				for i, a := range s.Assets {
					if a.Kind != "PF" && a.Kind != "NPS" && a.Kind != "EMPLOYER" {
						if fallback < 0 {
							fallback = i
						}
						weights[i] = (balances[i].liquid + balances[i].locked) * rate(a, m+1)
						weightTotal += weights[i]
					}
				}
				if weightTotal > 0 {
					for i, a := range s.Assets {
						contributions[i] += surplus * weights[i] / weightTotal / rate(a, m+1)
					}
				} else if fallback >= 0 {
					contributions[fallback] += surplus / rate(s.Assets[fallback], m+1)
				}
			}
		}
		if m%12 == 0 && s.RSUAnnual > 0 {
			grants = append(grants, grant{s.RSUAnnual * math.Pow(1+s.RSUGrantGrowth/100, float64(year)) / float64(s.RSUVestYears*12), s.RSUVestYears * 12})
		}
		for i := range grants {
			if grants[i].remaining > 0 {
				contributions[employer] += grants[i].monthly * s.RSUNetPercent / 100
				grants[i].remaining--
			}
		}
		monthlyInvestment := 0.0
		for i, a := range s.Assets {
			amount := contributions[i]
			invested += amount * rate(a, m+1)
			monthlyInvestment += amount * rate(a, m+1)
			if balances[i].released {
				balances[i].liquid += amount * a.AccessiblePercent / 100
				balances[i].locked += amount * (1 - a.AccessiblePercent/100)
			} else {
				balances[i].locked += amount
			}
		}
		if collect && len(points) > 0 && points[len(points)-1].Age == age {
			points[len(points)-1].MonthlyInvestment = monthlyInvestment
		}
	}
	return points, at, ok
}
