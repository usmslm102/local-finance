import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import {test} from 'node:test'
import ts from 'typescript'
const source = await readFile(new URL('../src/lib/fire-display.ts', import.meta.url), 'utf8')
const {outputText} = ts.transpileModule(source, {compilerOptions:{target:ts.ScriptTarget.ES2022,module:ts.ModuleKind.ES2022}})
const {fireNeedsFX,fireInputNeeds,fireStartingValues,fireAssetBreakdown,fireTargetProgress} = await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const asset = (overrides={}) => ({kind:'MARKET',name:'Fixture',currency:'INR',balance:100000,monthly:0,monthly_increase:0,return_percent:5,access_age:0,accessible_percent:100,...overrides})
const scenario = (overrides={}) => ({age:30,monthly_expense:40000,retirement_expense:60000,usd_inr:0,rsu_unvested_usd:0,rsu_annual_usd:0,invest_surplus:false,assets:[asset()],...overrides})

test('quick setup exposes conversion before querying when USD funding, balances or grants need it',()=>{
 for(const s of [scenario({assets:[asset({currency:'USD',balance:1})]}),scenario({assets:[asset({currency:'USD',balance:0,monthly:1000})]}),scenario({rsu_annual_usd:5000}),scenario({invest_surplus:true,assets:[asset({currency:'USD',balance:0})]})]){
  assert.equal(fireNeedsFX(s),true);assert.ok(fireInputNeeds(s).includes('an INR per USD rate'));assert.equal(fireStartingValues(s),null)
 }
 assert.equal(fireNeedsFX(scenario({assets:[asset({currency:'USD',balance:0})]})),false)
 assert.deepEqual(fireInputNeeds(scenario({age:0,monthly_expense:0,retirement_expense:0})),['your age','current living costs','retirement living costs'])
})
test('starting progress uses accessible assets in one current INR basis, excluding locked/unavailable portions',()=>{
 const s=scenario({usd_inr:80,assets:[asset(),asset({currency:'USD',balance:100,accessible_percent:50}),asset({balance:200000,access_age:60})]})
 assert.deepEqual(fireStartingValues(s),{total:308000,accessible:104000})
})
test('retirement breakdown uses one engine point and money basis, never current mixed with future values',()=>{
 const point={portfolio:2000000,accessible:1500000,today_value:1000000}
 const r={at_retirement:point,points:[{portfolio:123,accessible:0,today_value:123}]}
 assert.deepEqual(fireAssetBreakdown(r,false),{total:2000000,accessible:1500000,locked:500000,atRetirement:true})
 assert.deepEqual(fireAssetBreakdown(r,true),{total:1000000,accessible:750000,locked:250000,atRetirement:true})
})
test('unreachable and zero portfolios remain explicit without division by zero or an invented retirement snapshot',()=>{
 assert.deepEqual(fireAssetBreakdown({at_retirement:null,points:[{portfolio:0,accessible:0,today_value:0}]},true),{total:0,accessible:0,locked:0,atRetirement:false})
 assert.equal(fireAssetBreakdown({at_retirement:null,points:[]},true),null)
})

test('progress caps only the meter and retains the true above-target percentage',()=>{
 assert.deepEqual(fireTargetProgress(2000000,1000000),{meter:100,label:'200%'})
 assert.deepEqual(fireTargetProgress(500000,1000000),{meter:50,label:'50%'})
 assert.deepEqual(fireTargetProgress(0,0),{meter:0,label:'Unavailable'})
})
