<script setup>
import { computed, onMounted, ref } from 'vue'
import { getFinanceExchangeRates } from '@/services/finance'
import { financeError, financeDate } from '@/composables/useFinanceLedger'
import { downloadFinanceXlsx, openFinancePrint, writeFinancePrint } from '@/utils/financeDocuments'
import FasSelect from '@/components/ui/FasSelect.vue'
const snapshot=ref(null),loading=ref(false),error=ref(''),amount=ref('1'),from=ref('USD'),to=ref('IDR')
const load=async()=>{if(loading.value)return;loading.value=true;error.value='';try{snapshot.value=await getFinanceExchangeRates()}catch(e){snapshot.value=null;error.value=financeError(e)}finally{loading.value=false}}
const codes=computed(()=>Object.keys(snapshot.value?.rates || {}).sort())
const converted=computed(()=>{const n=Number(amount.value),r=snapshot.value?.rates;if(!r||!/^\d{1,12}(\.\d{1,6})?$/.test(amount.value)||!Number.isFinite(n)||!r[from.value]||!r[to.value])return null;return n/r[from.value]*r[to.value]})
const format=value=>new Intl.NumberFormat('id-ID',{maximumFractionDigits:6}).format(value)
const rows=computed(()=>codes.value.map(code=>[code,format(snapshot.value.rates[code]),format(snapshot.value.rates.IDR/snapshot.value.rates[code])]))
const exportReport=kind=>{try{const title='Referensi Kurs Internasional',description=`Sumber FreecurrencyAPI · Diambil ${financeDate(snapshot.value.fetched_at)} · Data akhir hari; bukan kurs pajak KMK, kurs bank, atau kurs eksekusi transfer.`;if(kind==='excel')downloadFinanceXlsx('PTFAS_Kurs_Referensi.xlsx',[[title],[description],['Mata uang','Per 1 USD','IDR per 1 unit'],...rows.value]);else writeFinancePrint(openFinancePrint(),title,description,['Mata uang','Per 1 USD','IDR per 1 unit'],rows.value)}catch(e){error.value=financeError(e)}}
onMounted(load)
</script>
<template>
 <section class="panel exchange-panel"><header class="panel-heading"><div><h2>Kurs internasional</h2><small>Referensi konversi mata uang · FreecurrencyAPI</small></div><div class="actions"><button :disabled="loading" @click="load">{{ loading?'Memuat…':'Perbarui kurs' }}</button><button :disabled="!snapshot||loading" @click="exportReport('excel')">Excel</button><button :disabled="!snapshot||loading" @click="exportReport('print')">Print / PDF</button></div></header>
 <div class="exchange-body"><p v-if="error" class="message error" role="alert">{{ error }}</p><template v-if="snapshot"><div class="form-grid"><label>Nominal konversi<input v-model="amount" inputmode="decimal" maxlength="20" placeholder="1" /></label><label>Dari mata uang<FasSelect v-model="from" aria-label="Dari mata uang"><option v-for="code in codes" :key="code" :value="code">{{ code }}</option></FasSelect></label><label>Ke mata uang<FasSelect v-model="to" aria-label="Ke mata uang"><option v-for="code in codes" :key="code" :value="code">{{ code }}</option></FasSelect></label><div class="conversion-result"><small>Estimasi hasil</small><strong>{{ converted === null ? 'Periksa nominal' : `${format(converted)} ${to}` }}</strong></div></div><small>Diambil {{ financeDate(snapshot.fetched_at) }}. Penyedia memakai data akhir hari; cache backend 6 jam. Ini bukan kurs pajak KMK atau kurs transaksi bank.</small><div class="exchange-highlights"><article v-for="code in ['USD','EUR','CNY','SGD','JPY','MYR']" :key="code"><span>1 {{ code }}</span><strong>{{ snapshot.rates[code] ? 'Rp '+format(snapshot.rates.IDR/snapshot.rates[code]) : 'Tidak tersedia' }}</strong></article></div></template><p v-else-if="loading" role="status">Mengambil referensi kurs…</p><p v-else class="muted">Kurs akan muncul setelah koneksi API tersedia.</p></div>
 </section>
</template>
<style scoped>
.exchange-body{padding:1.1rem}.conversion-result{padding:.7rem 1rem;border:1px solid var(--fas-accent-border);background:var(--fas-accent-soft);border-radius:.85rem;overflow-wrap:anywhere}.conversion-result strong{display:block;font-size:1.3rem;color:var(--fas-accent-text);font-variant-numeric:tabular-nums}.exchange-highlights{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.7rem;margin-top:1rem}.exchange-highlights article{padding:.8rem;border:1px solid var(--fas-border);border-radius:.85rem;background:var(--fas-overlay)}.exchange-highlights span{display:block;color:var(--fas-muted);font-size:.72rem}.exchange-highlights strong{display:block;margin-top:.3rem;font-variant-numeric:tabular-nums;font-size:.9rem}@media(max-width:600px){.exchange-highlights{grid-template-columns:1fr 1fr}}
</style>
