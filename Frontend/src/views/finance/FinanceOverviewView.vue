<script setup>
import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { getFinanceSalesSummary } from '@/services/finance'
import { getCustomers } from '@/services/customer'

const router = useRouter()
const activePage = 'overview'
const loading = ref(true)
const errorMessage = ref('')
const summary = ref({})
const customers = ref([])

const go = path => router.push(path)

const formatCurrency = value => new Intl.NumberFormat('id-ID', {
  style: 'currency',
  currency: 'IDR',
  maximumFractionDigits: 0,
}).format(Number(value || 0))

const customerActive = computed(() =>
  customers.value.filter(item => item.is_active).length,
)

const customerTaxComplete = computed(() =>
  customers.value.filter(item =>
    item.is_active &&
    String(item.tax_id_number || '').trim() &&
    String(item.tax_name || '').trim() &&
    String(item.tax_address || '').trim(),
  ).length,
)

const loadData = async () => {
  loading.value = true
  errorMessage.value = ''
  try {
    const [sales, customerRows] = await Promise.all([
      getFinanceSalesSummary({}),
      getCustomers(),
    ])
    summary.value = sales?.data || {}
    customers.value = Array.isArray(customerRows) ? customerRows : []
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Overview Finance belum dapat dimuat.'
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div class="finance-view">
    <header class="page-header">
      <div>
        <p class="kicker">Finance & Accounting</p>
        <h1>Overview Finance</h1>
        <p class="description">
          Ringkasan operasional Finance. Detail data dipisahkan ke halaman masing-masing supaya tidak perlu scroll terlalu panjang.
        </p>
      </div>
      <div class="actions">
        <button class="button button-secondary" :disabled="loading" @click="loadData">
          {{ loading ? 'Memuat...' : 'Refresh' }}
        </button>
      </div>
    </header>

    <FinanceNavigation />

    <div v-if="errorMessage" class="message error">{{ errorMessage }}</div>

    <section class="cards">
      <article class="card" @click="go('/finance/sales')">
        <span>Total Penjualan</span>
        <strong>{{ formatCurrency(summary.grand_total) }}</strong>
        <small>{{ Number(summary.invoice_count || 0) }} faktur</small>
      </article>
      <article class="card" @click="go('/finance/sales')">
        <span>Penjualan Bersih</span>
        <strong>{{ formatCurrency(summary.net_sales) }}</strong>
        <small>Sebelum pajak</small>
      </article>
      <article class="card" @click="go('/finance/customers')">
        <span>Customer Aktif</span>
        <strong>{{ customerActive }}</strong>
        <small>{{ customerTaxComplete }} data pajak lengkap</small>
      </article>
      <article class="card" @click="go('/finance/tax')">
        <span>PPN</span>
        <strong>{{ formatCurrency(summary.tax_amount) }}</strong>
        <small>DPP {{ formatCurrency(summary.tax_base_amount) }}</small>
      </article>
    </section>

    <section class="panel">
      <div class="panel-header">
        <div>
          <h2>Akses Cepat</h2>
          <p>Masuk langsung ke data yang sedang dibutuhkan Finance.</p>
        </div>
      </div>
      <div class="quick-grid">
        <button @click="go('/finance/analytics')"><strong>Analytics</strong><span>Statistik penjualan real-time</span></button>
        <button @click="go('/finance/sales')"><strong>Rekap Penjualan</strong><span>Faktur, DPP, PPN, PDF dan Excel</span></button>
        <button @click="go('/finance/customers')"><strong>Customer & Pajak</strong><span>Tambah toko, NPWP/NIK, PKP, TKU</span></button>
        <button @click="go('/finance/receivables')"><strong>Piutang</strong><span>Tahap berikutnya</span></button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.quick-grid {
  display: grid;
  grid-template-columns: repeat(4,minmax(0,1fr));
  gap: .7rem;
  padding: 1rem;
}
.quick-grid button {
  min-height: 7rem;
  border: 1px solid var(--fas-border);
  border-radius: .8rem;
  background: rgba(15,23,42,.45);
  padding: .8rem;
  color: inherit;
  text-align: left;
  cursor: pointer;
}
.quick-grid strong { display:block; color:var(--fas-text); font-size:.68rem; }
.quick-grid span { display:block; margin-top:.35rem; color:var(--fas-muted); font-size:.5rem; line-height:.85rem; }
@media(max-width:900px){.quick-grid{grid-template-columns:repeat(2,1fr)}}
@media(max-width:600px){.quick-grid{grid-template-columns:1fr}}
</style>

<style scoped>
.finance-view {
  min-height: 100%;
  padding: 1.35rem 1.5rem 2.5rem;
  color: var(--fas-text);
}
.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}
.kicker {
  margin: 0;
  color: var(--fas-accent-text);
  font-size: .58rem;
  font-weight: 850;
  letter-spacing: .1em;
  text-transform: uppercase;
}
h1 {
  margin: .3rem 0 0;
  font-size: 1.65rem;
  letter-spacing: -.03em;
}
.description {
  max-width: 48rem;
  margin: .45rem 0 0;
  color: var(--fas-muted);
  font-size: .66rem;
  line-height: 1.15rem;
}
.finance-nav {
  display: flex;
  flex-wrap: wrap;
  gap: .45rem;
  margin-bottom: 1rem;
}
.finance-nav button {
  min-height: 2.1rem;
  border: 1px solid var(--fas-border);
  border-radius: .58rem;
  background: rgba(15,23,42,.55);
  padding: 0 .68rem;
  color: var(--fas-muted);
  font-size: .52rem;
  font-weight: 800;
  cursor: pointer;
}
.finance-nav button.active {
  border-color: rgba(59,130,246,.5);
  background: rgba(37,99,235,.14);
  color: var(--fas-accent-text);
}
.panel {
  border: 1px solid var(--fas-border);
  border-radius: 1rem;
  background: rgba(10,17,31,.72);
  overflow: hidden;
}
.panel + .panel {
  margin-top: 1rem;
}
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: .9rem 1rem;
  border-bottom: 1px solid var(--fas-border);
}
.panel-header h2 {
  margin: 0;
  font-size: .9rem;
}
.panel-header p {
  margin: .24rem 0 0;
  color: #64748b;
  font-size: .53rem;
}
.cards {
  display: grid;
  grid-template-columns: repeat(4,minmax(0,1fr));
  gap: .7rem;
}
.card {
  min-height: 7.8rem;
  border: 1px solid var(--fas-border);
  border-radius: .9rem;
  background: linear-gradient(145deg,rgba(15,23,42,.88),rgba(10,17,31,.95));
  padding: .9rem;
}
.card span {
  color: var(--fas-muted);
  font-size: .56rem;
  font-weight: 700;
}
.card strong {
  display: block;
  margin-top: 1.15rem;
  font-size: 1.15rem;
}
.card small {
  display: block;
  margin-top: .35rem;
  color: #64748b;
  font-size: .49rem;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: .5rem;
}
.button {
  min-height: 2.35rem;
  border-radius: .65rem;
  padding: 0 .8rem;
  font-size: .56rem;
  font-weight: 800;
  cursor: pointer;
}
.button-primary {
  border: 1px solid rgba(59,130,246,.5);
  background: var(--fas-accent);
  color: white;
}
.button-secondary {
  border: 1px solid var(--fas-border);
  background: rgba(15,23,42,.6);
  color: var(--fas-text-soft);
}
.button-green {
  border: 1px solid rgba(16,185,129,.35);
  background: rgba(16,185,129,.1);
  color: #6ee7b7;
}
.button:disabled {
  cursor: not-allowed;
  opacity: .5;
}
.message {
  margin-bottom: 1rem;
  border-radius: .7rem;
  padding: .7rem .85rem;
  font-size: .6rem;
  font-weight: 700;
}
.error {
  border: 1px solid rgba(248,113,113,.25);
  background: rgba(127,29,29,.15);
  color: #fca5a5;
}
.success {
  border: 1px solid rgba(16,185,129,.24);
  background: rgba(6,78,59,.16);
  color: #6ee7b7;
}
.table-shell {
  max-height: 36rem;
  overflow: auto;
}
table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
}
thead {
  position: sticky;
  top: 0;
  z-index: 3;
}
th {
  border-bottom: 1px solid var(--fas-border);
  background: #0d1524;
  padding: .62rem .68rem;
  color: #748399;
  font-size: .46rem;
  font-weight: 850;
  letter-spacing: .05em;
  text-align: left;
  text-transform: uppercase;
  white-space: nowrap;
}
td {
  border-bottom: 1px solid rgba(51,65,85,.46);
  padding: .66rem .68rem;
  color: var(--fas-text-soft);
  font-size: .55rem;
  vertical-align: top;
}
td strong, td small {
  display: block;
}
td strong {
  color: var(--fas-text);
}
td small {
  margin-top: .15rem;
  color: #64748b;
  font-size: .46rem;
}
.input, select.input, textarea.input {
  width: 100%;
  min-height: 2.45rem;
  border: 1px solid var(--fas-border);
  border-radius: .65rem;
  outline: none;
  background: rgba(7,14,26,.82);
  padding: .55rem .7rem;
  color: var(--fas-text);
  font-size: .6rem;
}
textarea.input {
  min-height: 5rem;
  resize: vertical;
}
.filters {
  display: grid;
  grid-template-columns: minmax(0,1fr) 12rem 12rem;
  gap: .6rem;
  padding: .8rem 1rem;
}
.empty {
  padding: 3rem 1rem;
  color: var(--fas-muted);
  font-size: .62rem;
  text-align: center;
}
.modal-shell {
  position: fixed;
  z-index: 1500;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 1rem;
}
.modal-backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(2,6,23,.86);
}
.modal {
  position: relative;
  z-index: 2;
  width: min(96vw,70rem);
  max-height: 94vh;
  overflow: auto;
  border: 1px solid rgba(71,85,105,.8);
  border-radius: 1rem;
  background: #0b1220;
}
.modal-header {
  position: sticky;
  top: 0;
  z-index: 4;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--fas-border);
  background: #0b1220;
  padding: .8rem .9rem;
}
.modal-header h3 {
  margin: 0;
  font-size: .85rem;
}
.modal-body {
  padding: .9rem;
}
.form-grid {
  display: grid;
  grid-template-columns: repeat(2,minmax(0,1fr));
  gap: .7rem;
}
.form-grid label {
  display: grid;
  gap: .28rem;
}
.form-grid label > span {
  color: var(--fas-muted);
  font-size: .5rem;
  font-weight: 750;
}
.span-two {
  grid-column: span 2;
}
@media (max-width: 1100px) {
  .cards { grid-template-columns: repeat(2,minmax(0,1fr)); }
  .filters { grid-template-columns: 1fr 1fr; }
  .filters > :first-child { grid-column: span 2; }
}
@media (max-width: 720px) {
  .finance-view { padding: 1rem .85rem 2rem; }
  .page-header, .panel-header { align-items: stretch; flex-direction: column; }
  .cards, .filters, .form-grid { grid-template-columns: 1fr; }
  .filters > :first-child, .span-two { grid-column: span 1; }
}
</style>
