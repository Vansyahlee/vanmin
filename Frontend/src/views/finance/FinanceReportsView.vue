<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import FinanceExchangeRates from '@/components/finance/FinanceExchangeRates.vue'
import FasSelect from '@/components/ui/FasSelect.vue'
import { getFinanceReportPeriods,getFinanceSales,getFinanceExpenses,getFinanceCash,getFinanceReceivables,getFinanceTaxRegister,getFinanceProductTrend } from '@/services/finance'
import { financeToday,financeError,financeMoney,financeDate } from '@/composables/useFinanceLedger'
import { periodProfit,monthDates,reportMoney as money,reportCents,reportDecimal } from '@/utils/financeReporting'
import { downloadFinanceXlsx,openFinancePrint,writeFinancePrint } from '@/utils/financeDocuments'
import '@/assets/finance-ledger.css'
const definitions={
 SUMMARY:{label:'Ringkasan bulanan',headings:['Bulan','Penjualan sebelum PPN','Laba bersih manajerial','Kas masuk tercatat','Kas keluar tercatat','Kelengkapan biaya'],map:x=>[x.period,money(x.revenue),money(periodProfit(x)?.net),money(x.incoming),money(x.outgoing),x.full_month===false?'Rentang bukan bulan penuh':periodProfit(x)?'Dasar biaya terisi':'Belum lengkap']},
 SALES:{label:'Penjualan',fetch:getFinanceSales,headings:['Faktur','Tanggal','Customer','Penjualan sebelum PPN','PPN','Total','Status'],map:x=>[x.invoice_number,String(x.invoice_date).slice(0,10),x.store_name,financeMoney(Number(x.total)-Number(x.tax_amount)),financeMoney(x.tax_amount),financeMoney(x.total),x.status]},
 EXPENSES:{label:'Pengeluaran',fetch:getFinanceExpenses,headings:['Nomor','Tanggal','Jenis','Penerima','Uraian','Nominal','Metode','Status'],map:x=>[x.expense_number,x.expense_date,x.expense_kind,x.payee,x.description,money(x.amount),x.method,x.status]},
 CASH:{label:'Mutasi kas & bank',fetch:getFinanceCash,headings:['Waktu','Akun','Arah','Jenis','Pihak','Referensi','Nominal','Saldo berjalan'],map:x=>[financeDate(x.occurred_at),x.account,x.direction,x.kind,x.party,x.reference_number,money(x.amount),money(x.running_balance)]},
 RECEIVABLES:{label:'Piutang faktur periode — saldo saat ini',fetch:getFinanceReceivables,headings:['Faktur','Customer','Tanggal','Jatuh tempo','Total','Dibayar neto','Sisa saat ini','Status'],map:x=>[x.invoice_number,x.store_name,x.invoice_date,x.due_date,money(x.total),money(x.paid_amount),money(x.outstanding),x.payment_status]},
 TAX:{label:'Rekap pajak sumber',fetch:getFinanceTaxRegister,headings:['Faktur','Tanggal','Customer','NPWP','Mode','Penjualan sebelum PPN','DPP PPN','PPN'],map:x=>[x.invoice_number,x.invoice_date,x.store_name,x.buyer_tin,x.tax_mode,money(x.net_sales),money(x.tax_base),money(x.vat)]},
 TREND:{label:'Tren seluruh produk',fetch:getFinanceProductTrend,headings:['Produk','Kode','Kelompok','Jumlah faktur','Qty','Unit','Penjualan neto','Stok'],map:x=>[x.name,x.product_code,({FAST:'Terlaris',MEDIUM:'Sedang',UNSOLD:'Belum terjual'})[x.category],x.invoice_count,x.qty,x.unit,financeMoney(x.net_sales),x.stock]},
}
const initial=monthDates(financeToday().slice(0,7)),form=reactive({kind:'SUMMARY',...initial,query:''}),applied=ref(null),result=ref([]),loading=ref(false),error=ref(''),page=ref(0),snapshot=ref(null)
const definition=computed(()=>definitions[applied.value?.kind||form.kind]),headings=computed(()=>definition.value.headings),mapped=computed(()=>result.value.map(definition.value.map)),visible=computed(()=>mapped.value.slice(page.value*15,(page.value+1)*15)),pages=computed(()=>Math.max(1,Math.ceil(result.value.length/15)))
const sum=key=>reportDecimal(result.value.reduce((n,x)=>n+reportCents(x[key]||'0'),0n))
const summary=computed(()=>applied.value?.kind==='SUMMARY'?{revenue:sum('revenue'),incoming:sum('incoming'),outgoing:sum('outgoing'),profit:result.value.every(x=>periodProfit(x))?reportDecimal(result.value.reduce((n,x)=>n+reportCents(periodProfit(x).net),0n)):null}:null)
const range=mode=>{const y=financeToday().slice(0,4);Object.assign(form,mode==='year'?{date_from:y+'-01-01',date_to:y+'-12-31'}:monthDates(financeToday().slice(0,7)))}
const load=async()=>{
 if(loading.value)return;loading.value=true;error.value='';applied.value=null;result.value=[];snapshot.value=null;page.value=0
 const selected={...form};if(selected.kind==='SUMMARY')selected.query=''
 try{
  if(!selected.date_from||!selected.date_to||selected.date_from>selected.date_to)throw new Error('Pilih rentang tanggal yang valid.')
  if(selected.kind==='SUMMARY'){const r=await getFinanceReportPeriods(selected);result.value=r.data;snapshot.value=r}
  else {
   const items=[],seen=new Set();let expected=null
   for(let offset=0;;offset+=200){const r=await definitions[selected.kind].fetch({...selected,limit:200,offset,status:selected.kind==='EXPENSES'?'POSTED':'',sort:'frequency',category:''});const total=Number(r.meta?.total||0);if(expected===null)expected=total;if(expected!==total)throw new Error('Data berubah saat dimuat. Silakan ulangi.');for(const item of r.data||[]){if(seen.has(item.id))throw new Error('Urutan data berubah. Silakan ulangi.');seen.add(item.id);items.push(item)}if(items.length>=expected)break;if(!r.data?.length||offset>10000000)throw new Error('Data laporan belum lengkap.')}
   result.value=items
  }
  applied.value=selected
 }catch(e){result.value=[];error.value=financeError(e)}finally{loading.value=false}
}
const exportReport=kind=>{try{const title='Laporan Finance — '+definition.value.label,desc=`Periode ${applied.value.date_from} s.d. ${applied.value.date_to} · Cari: ${applied.value.query||'Semua'} · ${mapped.value.length} baris seluruh hasil · Dibuat ${financeDate(new Date())}. ${applied.value.kind==='SUMMARY'?'Laba manajerial memakai dasar biaya bulanan Finance; bukan laporan audit.':''}${applied.value.kind==='RECEIVABLES'?'Piutang adalah sisa faktur saat laporan diambil, bukan rekonstruksi saldo historis.':''}`;if(kind==='excel')downloadFinanceXlsx('PTFAS_Laporan_'+applied.value.kind+'.xlsx',[[title],[desc],headings.value,...mapped.value]);else writeFinancePrint(openFinancePrint(),title,desc,headings.value,mapped.value)}catch(e){error.value=financeError(e)}}
onMounted(load)
</script>
<template>
 <main class="finance-ledger"><header class="page-heading"><div><p class="eyebrow">Finance · Reports</p><h1>Pusat Laporan Finance</h1><p class="muted">Satu tempat untuk meninjau, mencetak, dan mengunduh seluruh hasil sesuai filter.</p></div></header><FinanceNavigation />
 <form class="panel" @submit.prevent="load"><fieldset :disabled="loading"><div class="filters"><label>Jenis laporan<FasSelect v-model="form.kind" aria-label="Jenis laporan"><option v-for="(item,key) in definitions" :key="key" :value="key">{{ item.label }}</option></FasSelect></label><label>Dari tanggal<input v-model="form.date_from" type="date" required /></label><label>Sampai tanggal<input v-model="form.date_to" type="date" required /></label><label v-if="form.kind!=='SUMMARY'">Cari pada laporan<input v-model="form.query" type="search" maxlength="200" placeholder="Faktur, customer, penerima, atau produk" /></label><div class="actions"><button class="primary" type="submit">{{ loading?'Menyiapkan seluruh data…':'Tampilkan laporan' }}</button><button type="button" @click="range('month')">Bulan ini</button><button type="button" @click="range('year')">Tahun ini</button></div></div></fieldset></form>
 <p v-if="error" class="message error" role="alert">{{ error }}</p>
 <template v-if="applied"><section v-if="summary" class="cards"><article class="card"><span>Penjualan sebelum PPN</span><strong>{{ money(summary.revenue) }}</strong></article><article class="card"><span>Laba bersih manajerial</span><strong>{{ money(summary.profit) }}</strong><small>Semua bulan harus lengkap dan memakai rentang bulan penuh.</small></article><article class="card"><span>Perubahan kas tercatat</span><strong>{{ money(reportDecimal(reportCents(summary.incoming)-reportCents(summary.outgoing))) }}</strong><small>Kas masuk {{ money(summary.incoming) }} · Keluar {{ money(summary.outgoing) }}</small></article></section>
 <p v-if="summary" class="message">Saldo awal tercatat {{ money(snapshot.cash_opening) }} · Saldo akhir tercatat {{ money(reportDecimal(reportCents(snapshot.cash_opening)+reportCents(summary.incoming)-reportCents(summary.outgoing))) }}. Belum merupakan saldo bank aktual atau rekonsiliasi bank.</p>
 <section class="panel"><header class="panel-heading"><div><h2>{{ definition.label }}</h2><small>{{ applied.date_from }} s.d. {{ applied.date_to }} · Cari: {{ applied.query||'Semua' }}</small></div><div class="actions"><button :disabled="loading" @click="exportReport('excel')">Excel seluruh hasil</button><button :disabled="loading" @click="exportReport('print')">Print / PDF seluruh hasil</button></div></header><div class="table-scroll"><table><thead><tr><th v-for="heading in headings" :key="heading">{{ heading }}</th></tr></thead><tbody><tr v-for="(cells,index) in visible" :key="index"><td v-for="(cell,c) in cells" :key="c">{{ cell }}</td></tr><tr v-if="!visible.length"><td class="empty" :colspan="headings.length">Tidak ada data sesuai filter.</td></tr></tbody></table></div><footer class="pagination"><span>{{ result.length }} baris · Halaman {{ page+1 }} / {{ pages }} · 15 baris per halaman</span><div class="actions"><button :disabled="page===0||loading" @click="page--">Sebelumnya</button><button :disabled="page+1>=pages||loading" @click="page++">Berikutnya</button></div></footer></section>
 </template><FinanceExchangeRates />
 </main>
</template>
