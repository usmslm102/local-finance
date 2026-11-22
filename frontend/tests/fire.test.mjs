import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {test} from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/lib/fire.ts', import.meta.url), 'utf8')
const {outputText} = ts.transpileModule(source, {compilerOptions: {target: ts.ScriptTarget.ES2022, module: ts.ModuleKind.ES2022}})
const {fireDefaults, loadFireScenario, fireAge, fireChartPoints} = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const baseline = {monthly_expense: 60000, monthly_salary: 200000, monthly_india: 45000, monthly_us_inr: 25000, monthly_deposits: 5000, monthly_other: 1000, india_balance: 1200000, usd_balance: 4000}
test('FIRE prefills retain INR contribution units, distinct snapshots and explicit unknown age/FX', () => {
 const s = fireDefaults(baseline)
 assert.equal(s.age, 0); assert.equal(s.usd_inr, 0)
 assert.equal(s.assets[1].balance, 4000); assert.equal(s.assets[1].monthly, 25000)
 assert.equal(s.assets[1].monthly, fireDefaults(baseline, 80).assets[1].monthly)
 assert.equal(s.assets[0].balance, 1200000); assert.equal(s.assets[4].kind, 'PF')
 assert.equal(s.assets[5].accessible_percent, 0); assert.equal(s.rsu_unvested_usd, 0)
})
test('saved FIRE scenarios require complete finite fields and preserve all manual inputs', () => {
 const s = fireDefaults(baseline, 80); s.age = 26; s.assets[4].balance = 1200000; s.rsu_annual_usd = 15000
 assert.deepEqual(loadFireScenario(JSON.stringify(s)), s)
 for (const raw of ['bad', '{}', 'null', JSON.stringify({...s, horizon: null}), JSON.stringify({...s, assets: [{name: 'broken'}]})]) assert.equal(loadFireScenario(raw), null)
})
test('retirement ages display month precision without rounding to a different year', () => {
 assert.equal(fireAge(40), '40 years'); assert.equal(fireAge(40 + 11 / 12), '40 years 11 months')
})

test('charts retain exact retirement marker, deflate once and end the initial target after retirement', () => {
 const point = {age:40,portfolio:200,accessible:100,today_value:100,target:180,invested:10,monthly_investment:0,monthly_expense:5,retired:true}
 const result = {retirement_age:40,today_target:90,points:[point,{...point,age:41},{...point,age:42,portfolio:0,accessible:0,today_value:0}],warnings:[]}
 const real = fireChartPoints(result,true)
 assert.equal(real[0].portfolio,100); assert.equal(real[0].accessible,50); assert.equal(real[0].target,90)
 assert.equal(real[1].target,null); assert.equal(real[2].accessible,0)
 assert.equal(fireChartPoints(result,false)[0].portfolio,200)
})
