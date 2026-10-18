<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useModalDialog } from '@/composables/useModalDialog'
import FasSelect from '@/components/ui/FasSelect.vue'
import CompanyCalculator from '@/components/finance/CompanyCalculator.vue'
import FinanceExchangeRates from '@/components/finance/FinanceExchangeRates.vue'
import {
  getFinanceReportPeriods,
  saveFinanceProfitPeriod,
  getFinanceReportingHistory,
} from '@/services/finance'
import {
  financeToday,
  financeError,
  financeDate,
} from '@/composables/useFinanceLedger'
import {
  monthDates,
  periodProfit,
  profitRows,
  reportMoney as money,
  reportPercent,
} from '@/utils/financeReporting'
import {
  downloadFinanceXlsx,
  openFinancePrint,
  writeFinancePrint,
} from '@/utils/financeDocuments'
import '@/assets/finance-ledger.css'
const month = ref(financeToday().slice(0, 7)),
  applied = ref(''),
  row = ref(null),
  loading = ref(false),
  busy = ref(false),
  error = ref(''),
  message = ref(''),
  history = ref([])
const calculatorOpen = ref(false),
  calculatorTarget = ref('hpp')
const historyOpen = ref(false)
const historyBusy = ref(false)
const historyError = ref('')
const adjustmentsOpen = ref(false)
const closeHistory = () => { historyOpen.value = false }
useModalDialog(() => (historyOpen.value ? '#profit-history-dialog' : ''), closeHistory)
const applyCalculator = (amount) => {
  if (busy.value || loading.value || !row.value) return
  form[calculatorTarget.value] = amount
  message.value =
    'Nominal kalkulator diterapkan. Simpan perubahan untuk memperbarui laporan.'
}
const fields = [
  ['hpp', 'HPP barang terjual'],
  ['salary_expense', 'Beban gaji periode'],
  ['operating_expense', 'Beban operasional selain gaji'],
  ['income_tax', 'Beban pajak penghasilan'],
  ['other_income', 'Pendapatan lain'],
  ['owner_percent', 'Simulasi bagian owner (%)'],
]
const form = reactive({
  hpp: '',
  salary_expense: '',
  operating_expense: '',
  income_tax: '',
  other_income: '',
  owner_percent: '',
  notes: '',
})
const saved = ref(''),
  dirty = computed(() => JSON.stringify(form) !== saved.value),
  profit = computed(() => (row.value ? periodProfit(row.value) : null)),
  preview = computed(() =>
    row.value ? periodProfit({ ...row.value, settings: form }) : null,
  )
const load = async (target = month.value) => {
  if (loading.value || busy.value) return
  loading.value = true
  error.value = ''
  history.value = []
  row.value = null
  try {
    const result = await getFinanceReportPeriods(monthDates(target))
    row.value = result.data?.[0]
    if (!row.value) throw new Error('Laporan periode ini belum tersedia.')
    applied.value = target
    for (const key of Object.keys(form))
      form[key] = String(row.value.settings?.[key] ?? '')
    saved.value = JSON.stringify(form)
  } catch (e) {
    error.value = financeError(e)
  } finally {
    loading.value = false
  }
}
const save = async () => {
  if (!row.value || busy.value || loading.value) return
  error.value = ''
  message.value = ''
  if (!preview.value || form.notes.trim().length < 5) {
    error.value =
      'Isi seluruh nominal (0 jika nihil), persentase 0–100, dan dasar perhitungan.'
    return
  }
  busy.value = true
  try {
    await saveFinanceProfitPeriod(applied.value, {
      revision: row.value.revision,
      settings: { ...form },
    })
    message.value = 'Perubahan dasar laba telah disimpan.'
    busy.value = false
    await load(applied.value)
  } catch (e) {
    error.value = financeError(e)
  } finally {
    busy.value = false
  }
}
const showHistory = async () => {
  if (!applied.value) return
  historyOpen.value = true
  historyBusy.value = true
  historyError.value = ''
  history.value = []
  try {
    const response = await getFinanceReportingHistory({
      kind: 'PROFIT',
      reference: applied.value + '-01',
    })
    history.value = Array.isArray(response.data) ? response.data : []
  } catch (e) {
    historyError.value = financeError(e)
  } finally {
    historyBusy.value = false
  }
}
const exportHistory = (kind) => {
  try {
    const title = 'Riwayat dasar laba — ' + applied.value,
      rows = history.value.map((x) => [
        financeDate(x.changed_at),
        x.actor_name,
        x.after_data?.revision ?? '—',
        ...[
          'hpp',
          'salary_expense',
          'operating_expense',
          'income_tax',
          'other_income',
        ].map((key) => money(x.after_data?.settings?.[key])),
        x.after_data?.settings?.owner_percent + '%',
        x.after_data?.settings?.notes,
      ]),
      headings = [
        'Waktu',
        'Petugas',
        'Revisi',
        'HPP',
        'Gaji',
        'Operasional',
        'Pajak',
        'Pendapatan lain',
        'Owner %',
        'Dasar',
      ]
    if (kind === 'excel')
      downloadFinanceXlsx('PTFAS_Riwayat_Laba.xlsx', [
        [title],
        ['Maksimal 50 perubahan terakhir'],
        headings,
        ...rows,
      ])
    else
      writeFinancePrint(
        openFinancePrint(),
        title,
        'Maksimal 50 perubahan terakhir',
        headings,
        rows,
      )
  } catch (e) {
    error.value = financeError(e)
  }
}
const documentRows = computed(() => (row.value ? profitRows(row.value) : []))
const exportReport = (kind) => {
  try {
    const title = 'Laba Rugi Manajerial — ' + applied.value,
      desc = `Penjualan faktur aktif sebelum PPN; biaya periode diisi Finance. ${profit.value ? 'Dasar perhitungan tersedia' : 'BELUM LENGKAP — laba belum dapat ditentukan'}. Bukan laporan audit atau persetujuan dividen. Dasar: ${row.value.settings?.notes || 'Belum diisi'}`
    if (kind === 'excel')
      downloadFinanceXlsx('PTFAS_Laba_' + applied.value + '.xlsx', [
        [title],
        [desc],
        ['Komponen', 'Nominal', '% penjualan'],
        ...documentRows.value,
      ])
    else
      writeFinancePrint(
        openFinancePrint(),
        title,
        desc,
        ['Komponen', 'Nominal', '% penjualan'],
        documentRows.value,
        { orientation: 'portrait' },
      )
  } catch (e) {
    error.value = financeError(e)
  }
}
onMounted(() => load())
</script>
<template>
  <main class="finance-ledger profit-page">
    <header class="profit-heading">
      <div>
        <p class="eyebrow">Finance / Profitabilitas</p>
        <h1>Laba &amp; Rugi</h1>
        <p class="profit-subtitle">Ringkasan hasil usaha berdasarkan periode laporan.</p>
      </div>
      <RouterLink class="profit-action-link" to="/finance/expenses">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" aria-hidden="true"><path d="M12 5v14m-7-7h14"/></svg>
        Catat pengeluaran
      </RouterLink>
    </header>

    <form class="profit-toolbar" @submit.prevent="load()">
      <label>
        <span>Periode laporan</span>
        <input v-model="month" type="month" required :disabled="loading || busy" />
      </label>
      <button class="primary" type="submit" :disabled="loading || busy">
        {{ loading ? 'Memuat…' : 'Tampilkan' }}
      </button>
      <div class="profit-toolbar-spacer"></div>
      <button type="button" :disabled="!row || dirty || loading" @click="exportReport('excel')">Excel</button>
      <button type="button" :disabled="!row || dirty || loading" @click="exportReport('print')">Print / PDF</button>
    </form>

    <p v-if="error" class="message error" role="alert">{{ error }}</p>
    <p v-if="message" class="message" role="status">{{ message }}</p>
    <p v-if="loading && !row" class="profit-loading" role="status">Menyiapkan laporan…</p>

    <template v-if="row">
      <section class="profit-kpis" aria-label="Ringkasan laba">
        <article class="profit-kpi">
          <span>Penjualan sebelum PPN</span>
          <strong>{{ money(row.revenue) }}</strong>
          <small>{{ row.invoice_count }} faktur aktif</small>
        </article>
        <article class="profit-kpi">
          <span>Harga pokok barang</span>
          <strong>{{ money(row.settings?.hpp) }}</strong>
          <small>HPP periode {{ applied }}</small>
        </article>
        <article class="profit-kpi" :class="{ 'profit-kpi-negative': profit && Number(profit.net) < 0 }">
          <span>{{ profit && Number(profit.net) < 0 ? 'Rugi bersih' : 'Laba bersih' }}</span>
          <strong>{{ money(profit?.net) }}</strong>
          <small>Margin {{ profit ? reportPercent(profit.net, row.revenue) : '—' }}</small>
        </article>
        <article class="profit-kpi">
          <span>Simulasi alokasi owner</span>
          <strong>{{ money(profit?.owner) }}</strong>
          <small>Bukan dana yang otomatis ditransfer</small>
        </article>
      </section>

      <div class="profit-workspace" :class="{ 'calculator-open': calculatorOpen }">
        <div class="profit-content">
          <section class="profit-panel">
            <header class="profit-panel-head">
              <div>
                <h2>Rincian Laba &amp; Rugi</h2>
                <p>Periode {{ applied }}</p>
              </div>
              <span class="profit-state" :class="{ 'profit-state-warn': !profit }">{{ profit ? 'Tersedia' : 'Belum lengkap' }}</span>
            </header>
            <div class="table-scroll">
              <table class="profit-table">
                <thead><tr><th>Komponen</th><th class="money">Nominal</th><th class="money">% penjualan</th></tr></thead>
                <tbody>
                  <tr v-for="[label, amount, ratio] in documentRows" :key="label">
                    <td>{{ label }}</td>
                    <td class="money">{{ amount }}</td>
                    <td class="money">{{ ratio }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <p v-if="!profit" class="profit-footnote">Beberapa komponen biaya periode belum dilengkapi.</p>
          </section>

          <section class="profit-dual">
            <article class="profit-panel profit-compact-panel">
              <div class="profit-mini-head">
                <div><h2>Pembayaran Tercatat</h2><p>Mutasi kas dalam periode laporan</p></div>
                <RouterLink to="/finance/expenses">Lihat pengeluaran →</RouterLink>
              </div>
              <div class="profit-paid-list">
                <div><span>Gaji dibayar</span><strong>{{ money(row.salary_paid) }}</strong></div>
                <div><span>Supplier dibayar</span><strong>{{ money(row.supplier_paid) }}</strong></div>
                <div><span>Operasional dibayar</span><strong>{{ money(row.general_paid) }}</strong></div>
              </div>
            </article>
            <article class="profit-panel profit-compact-panel profit-edit-card">
              <div class="profit-mini-head"><div><h2>Dasar Laporan</h2><p>Revisi {{ row.revision || 0 }}</p></div></div>
              <p>{{ row.settings?.notes || 'Belum ada keterangan perhitungan.' }}</p>
              <div class="profit-edit-actions">
                <button type="button" @click="adjustmentsOpen = !adjustmentsOpen">{{ adjustmentsOpen ? 'Tutup penyesuaian' : 'Ubah dasar laporan' }}</button>
                <button type="button" :disabled="busy || loading" @click="showHistory">Riwayat perubahan</button>
              </div>
            </article>
          </section>

          <section v-if="adjustmentsOpen" class="profit-panel profit-adjustments">
            <header class="profit-panel-head">
              <div><h2>Penyesuaian Dasar Laba</h2><p>Perubahan nilai akan tercatat dalam riwayat.</p></div>
              <button type="button" @click="adjustmentsOpen = false" aria-label="Tutup penyesuaian">Tutup</button>
            </header>
            <form @submit.prevent="save">
              <fieldset :disabled="busy || loading">
                <div class="profit-fields">
                  <label v-for="[key, label] in fields" :key="key">
                    <span>{{ label }}</span>
                    <input v-model="form[key]" required inputmode="decimal" maxlength="19" placeholder="0" />
                  </label>
                  <label class="profit-span-two">
                    <span>Dasar pencatatan / referensi</span>
                    <textarea v-model="form.notes" required minlength="5" maxlength="2000" rows="3" placeholder="Referensi biaya periode…" />
                  </label>
                </div>
                <div class="profit-calc-row">
                  <label>
                    <span>Isi dari kalkulator</span>
                    <FasSelect v-model="calculatorTarget" :disabled="busy || loading" aria-label="Komponen kalkulator">
                      <option v-for="[key, label] in fields" :key="key" :value="key">{{ label }}</option>
                    </FasSelect>
                  </label>
                </div>
                <div v-if="dirty" class="profit-preview">
                  <span>Pratinjau laba bersih</span>
                  <strong>{{ money(preview?.net) }}</strong>
                </div>
                <footer class="profit-save-row">
                  <button class="primary" type="submit" :disabled="!dirty || busy || loading">{{ busy ? 'Menyimpan…' : 'Simpan perubahan' }}</button>
                  <span>Terakhir disimpan {{ row.updated_at ? financeDate(row.updated_at) : '—' }}</span>
                </footer>
              </fieldset>
            </form>
          </section>

          <FinanceExchangeRates />
        </div>
        <CompanyCalculator @use="applyCalculator" @toggle="calculatorOpen = $event" />
      </div>
    </template>

    <Teleport to="body">
      <div v-if="historyOpen" class="profit-modal-backdrop" @click.self="closeHistory">
        <section id="profit-history-dialog" class="finance-ledger profit-history-dialog" role="dialog" aria-modal="true" aria-labelledby="profit-history-title">
          <header class="profit-dialog-head">
            <div><h2 id="profit-history-title">Riwayat Perubahan</h2><p>Dasar laba periode {{ applied }}</p></div>
            <button type="button" aria-label="Tutup riwayat" @click="closeHistory">Tutup</button>
          </header>
          <div class="profit-dialog-content">
            <p v-if="historyBusy" class="profit-empty">Memuat riwayat…</p>
            <p v-else-if="historyError" class="message error" role="alert">{{ historyError }}</p>
            <p v-else-if="!history.length" class="profit-empty">Belum ada perubahan pada periode ini.</p>
            <div v-else class="table-scroll">
              <table class="profit-history-table">
                <thead><tr><th>Waktu</th><th>Petugas</th><th>Revisi</th><th>Catatan</th></tr></thead>
                <tbody>
                  <tr v-for="item in history" :key="item.id">
                    <td>{{ financeDate(item.changed_at) }}</td>
                    <td>{{ item.actor_name || '—' }}</td>
                    <td>{{ item.after_data?.revision ?? '—' }}</td>
                    <td>{{ item.after_data?.settings?.notes || '—' }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
          <footer class="profit-dialog-actions">
            <button type="button" :disabled="historyBusy || !history.length" @click="exportHistory('excel')">Export Excel</button>
            <button type="button" :disabled="historyBusy || !history.length" @click="exportHistory('print')">Print / PDF</button>
          </footer>
        </section>
      </div>
    </Teleport>
  </main>
</template>
<style scoped>
.profit-page { --profit-radius: 1.2rem; padding: 1.65rem clamp(1rem,2.3vw,2.1rem) 3rem; }
.profit-heading {display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap;margin-bottom:1.3rem}
.profit-heading h1 {font-size:clamp(1.55rem,2.4vw,2rem);margin:.25rem 0;font-weight:780;letter-spacing:-.04em}
.profit-subtitle{color:var(--fas-muted);margin:.25rem 0 0;font-size:.82rem}
.profit-page .profit-action-link{display:inline-flex;align-items:center;gap:.5rem;padding:.8rem 1rem;border-radius:.95rem;background:var(--fas-accent);color:#fff;font-weight:730;font-size:.8rem;text-decoration:none;white-space:nowrap}
.profit-action-link svg{width:1rem;height:1rem}
.profit-toolbar {display:flex;align-items:end;gap:.65rem;flex-wrap:wrap;padding:1rem;border:1px solid var(--fas-border);background:var(--fas-card);border-radius:var(--profit-radius);margin-bottom:1.05rem}
.profit-toolbar label {display:grid;gap:.4rem;font-size:.73rem;color:var(--fas-text-soft);font-weight:700}
.profit-toolbar input{min-width:11.5rem}
.profit-toolbar-spacer{flex:1}
.profit-kpis {display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:.8rem;margin:1.15rem 0}
.profit-kpi{min-height:9.2rem;border:1px solid var(--fas-border);background:var(--fas-card);border-radius:var(--profit-radius);padding:1.1rem 1.2rem;display:flex;flex-direction:column;gap:.8rem;box-shadow:var(--fas-shadow-soft)}
.profit-kpi>span{font-size:.77rem;font-weight:650;color:var(--fas-muted)}
.profit-kpi>strong{font-size:clamp(1.15rem,2vw,1.65rem);font-weight:800;letter-spacing:-.035em;line-height:1.3;overflow-wrap:anywhere;color:var(--fas-text)}
.profit-kpi small{margin-top:auto;font-size:.71rem}
.profit-kpi-negative>strong{color:var(--fas-danger,#d44f57)}
.profit-workspace{display:grid;grid-template-columns:minmax(0,1fr) 3.25rem;gap:.8rem;align-items:start}
.profit-workspace.calculator-open{grid-template-columns:minmax(0,1fr) minmax(17rem,20rem)}
.profit-content{min-width:0;display:grid;gap:1rem}
.profit-panel{background:var(--fas-card);border:1px solid var(--fas-border);border-radius:var(--profit-radius);overflow:hidden;min-width:0;box-shadow:var(--fas-shadow-soft)}
.profit-panel-head{display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap;padding:1.1rem 1.2rem;border-bottom:1px solid var(--fas-border)}
.profit-panel-head h2,.profit-mini-head h2{font-size:1rem;font-weight:770;margin:0;color:var(--fas-text)}
.profit-panel-head p,.profit-mini-head p{font-size:.75rem;color:var(--fas-muted);margin:.35rem 0 0}
.profit-state{border-radius:999px;background:var(--fas-success-soft,var(--fas-accent-soft));color:var(--fas-success,var(--fas-accent-text));padding:.4rem .7rem;font-size:.7rem;font-weight:700}
.profit-state-warn{background:var(--fas-warning-soft,var(--fas-accent-soft));color:var(--fas-warning,var(--fas-accent-text))}
.profit-table{width:100%;min-width:32rem!important}
.profit-table th{white-space:nowrap}
.profit-table tbody tr:nth-child(3),.profit-table tbody tr:nth-child(10){background:var(--fas-accent-soft);font-weight:750}
.profit-footnote{padding:0 1.2rem;color:var(--fas-muted);font-size:.75rem}
.profit-dual{display:grid;grid-template-columns:1fr 1fr;gap:1rem}
.profit-compact-panel{padding:1.2rem}
.profit-mini-head{display:flex;align-items:start;justify-content:space-between;gap:.7rem;flex-wrap:wrap}
.profit-mini-head a{color:var(--fas-accent-text);font-size:.74rem;text-decoration:none;font-weight:700}
.profit-paid-list{margin-top:1rem;display:grid;gap:.85rem}
.profit-paid-list>div{display:flex;gap:1rem;justify-content:space-between;align-items:center;font-size:.8rem}
.profit-paid-list span{color:var(--fas-muted)}
.profit-paid-list strong{font-size:.89rem;color:var(--fas-text);text-align:right}
.profit-edit-card>p{font-size:.8rem;color:var(--fas-text-soft);line-height:1.65;margin:.85rem 0;min-height:2.5rem;overflow-wrap:anywhere}
.profit-edit-actions{display:flex;gap:.55rem;flex-wrap:wrap}
.profit-adjustments fieldset{border:0;margin:0;padding:1.2rem;min-width:0}
.profit-fields{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:1rem}
.profit-fields label,.profit-calc-row label{display:grid;gap:.42rem;font-size:.78rem;color:var(--fas-text-soft);font-weight:690;min-width:0}
.profit-fields input,.profit-fields textarea{width:100%;min-width:0}
.profit-span-two{grid-column:1/-1}
.profit-calc-row{margin-top:1rem;max-width:23rem}
.profit-preview{display:flex;align-items:center;justify-content:space-between;gap:1rem;background:var(--fas-accent-soft);border:1px solid var(--fas-accent-border);border-radius:.8rem;padding:.8rem 1rem;margin-top:1rem;font-size:.76rem}
.profit-preview strong{font-size:1.07rem;overflow-wrap:anywhere}
.profit-save-row{display:flex;align-items:center;gap:.8rem;flex-wrap:wrap;margin-top:1rem}
.profit-save-row span{color:var(--fas-muted);font-size:.72rem}
.profit-loading,.profit-empty{display:grid;place-items:center;min-height:10rem;color:var(--fas-muted);font-size:.85rem;text-align:center}
.profit-modal-backdrop{position:fixed;inset:0;z-index:2000;background:rgba(12,18,32,.58);display:flex;align-items:center;justify-content:center;padding:1.1rem;backdrop-filter:blur(5px)}
.profit-history-dialog{padding:0!important;background:var(--fas-card);color:var(--fas-text);border:1px solid var(--fas-border);border-radius:1.2rem;box-shadow:0 24px 80px rgba(0,0,0,.23);width:min(100%,56rem);max-height:min(85vh,49rem);display:flex;flex-direction:column;overflow:hidden}
.profit-dialog-head{display:flex;justify-content:space-between;align-items:center;gap:1rem;padding:1.15rem 1.3rem;border-bottom:1px solid var(--fas-border)}
.profit-dialog-head h2{font-size:1.1rem;font-weight:760;margin:0}
.profit-dialog-head p{margin:.2rem 0 0;font-size:.75rem;color:var(--fas-muted)}
.profit-dialog-content{min-height:0;overflow:auto}
.profit-history-table{width:100%;min-width:35rem}
.profit-dialog-actions{display:flex;justify-content:flex-end;gap:.6rem;padding:.9rem 1.3rem;border-top:1px solid var(--fas-border)}
@media (max-width:1199px){.profit-workspace,.profit-workspace.calculator-open{grid-template-columns:minmax(0,1fr)}}
@media (max-width:950px){.profit-kpis{grid-template-columns:repeat(2,minmax(0,1fr))}.profit-dual{grid-template-columns:1fr}}
@media (max-width:560px){.profit-page{padding:1rem .8rem 2rem}.profit-kpis{gap:.55rem}.profit-kpi{padding:.9rem;min-height:8rem}.profit-kpi>strong{font-size:1.13rem}.profit-toolbar-spacer{display:none}.profit-toolbar button{flex:1}.profit-fields{grid-template-columns:1fr}.profit-span-two{grid-column:auto}}
</style>
