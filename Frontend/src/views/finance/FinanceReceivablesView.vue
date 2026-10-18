<script setup>
import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import { companyDocumentHTML } from '@/services/company'
import { downloadFinanceXlsx } from '@/utils/financeDocuments'
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import FasIcon from '@/components/ui/FasIcon.vue'
import FasSelect from '@/components/ui/FasSelect.vue'
import { useModalDialog } from '@/composables/useModalDialog'
import { getFinanceReceivables } from '@/services/finance'

const invoiceReturnStorageKey = 'fas.receivables.invoice-return'
const takeInvoiceReturnState = () => {
  try {
    const raw = sessionStorage.getItem(invoiceReturnStorageKey)
    sessionStorage.removeItem(invoiceReturnStorageKey)
    const state = raw ? JSON.parse(raw) : null
    if (!state || Date.now() - Number(state.savedAt) > 30 * 60 * 1000) return null
    if (!state.filters || typeof state.filters !== 'object') return null
    return state
  } catch { return null }
}
const invoiceReturnState = takeInvoiceReturnState()
const router = useRouter()
const loading = ref(false)
const exporting = ref(false)
const errorMessage = ref('')
const rows = ref([])
const summary = ref({})
const agingRows = ref([])
const meta = ref({ total: 0, limit: 50, offset: Math.max(0, Number(invoiceReturnState?.offset) || 0), as_of: '' })
const selected = ref(null)
const updatedAt = ref(null)
const filters = reactive({ query: '', date_from: '', date_to: '', status: 'OPEN', aging: 'ALL', ...(invoiceReturnState?.filters || {}) })
const applied = ref({ ...filters })
let requestID = 0
let refreshTimer = null
let searchTimer = null
let disposed = false

const navigation = [
  ['overview', 'Overview', '/finance'],
  ['analytics', 'Analytics', '/finance/analytics'],
  ['sales', 'Penjualan', '/finance/sales'],
  ['customers', 'Customer', '/finance/customers'],
  ['receivables', 'Piutang', '/finance/receivables'],
  ['payments', 'Pembayaran', '/finance/payments'],
  ['expenses', 'Pengeluaran', '/finance/expenses'],
  ['cash', 'Kas & Bank', '/finance/cash'],
  ['profit', 'Laba', '/finance/profit'],
  ['tax', 'Pajak', '/finance/tax'],
  ['reports', 'Laporan', '/finance/reports'],
]

const statuses = [
  ['OPEN', 'Piutang aktif'],
  ['ALL', 'Semua tagihan'],
  ['NOT_DUE', 'Belum jatuh tempo'],
  ['DUE', 'Jatuh tempo hari ini'],
  ['OVERDUE', 'Terlambat'],
  ['PARTIAL', 'Dibayar sebagian'],
  ['PAID', 'Lunas'],
  ['UNSCHEDULED', 'Jatuh tempo belum diatur'],
]

const ageOptions = [
  ['NOT_DUE', 'Belum jatuh tempo'],
  ['DUE', 'Jatuh tempo hari ini'],
  ['1_7', 'Terlambat 1–7 hari'],
  ['8_14', 'Terlambat 8–14 hari'],
  ['15_30', 'Terlambat 15–30 hari'],
  ['OVER_30', 'Terlambat >30 hari'],
  ['UNSCHEDULED', 'Jatuh tempo belum diatur'],
]

const dueLabels = {
  NOT_DUE: 'Belum jatuh tempo',
  DUE: 'Jatuh tempo hari ini',
  OVERDUE: 'Terlambat',
  UNSCHEDULED: 'Belum diatur',
  PAID: 'Lunas',
}

const money = value => new Intl.NumberFormat('id-ID', {
  style: 'currency', currency: 'IDR', minimumFractionDigits: 0, maximumFractionDigits: 2,
}).format(Number(value || 0))

const number = value => new Intl.NumberFormat('id-ID').format(Number(value || 0))
const date = value => value ? new Intl.DateTimeFormat('id-ID', {
  dateStyle: 'medium', timeZone: 'Asia/Jakarta',
}).format(new Date(value.length === 10 ? `${value}T12:00:00+07:00` : value)) : 'Belum diatur'

const periodLabel = computed(() => {
  if (!applied.value.date_from && !applied.value.date_to) {
    return 'Seluruh tanggal faktur'
  }
  return `${applied.value.date_from ? date(applied.value.date_from) : 'Awal data'} – ${applied.value.date_to ? date(applied.value.date_to) : 'Sekarang'}`
})

const aging = computed(() => ageOptions.map(([key, label]) => {
  const row = agingRows.value.find(item => item.key === key)
  return { key, label, amount: Number(row?.amount || 0), count: Number(row?.count || 0) }
}))

const pages = computed(() => Math.max(1, Math.ceil(Number(meta.value.total) / meta.value.limit)))
const currentPage = computed(() => Math.floor(meta.value.offset / meta.value.limit) + 1)
const canReport = computed(() => !loading.value && !exporting.value && !errorMessage.value && meta.value.total > 0)

const load = async (reset = false) => {
  if (reset) {
    meta.value.offset = 0
  }
  if (filters.date_from && filters.date_to && filters.date_from > filters.date_to) {
    errorMessage.value = 'Tanggal awal tidak boleh melebihi tanggal akhir'
    return
  }
  const id = ++requestID
  const snapshot = { ...filters }
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await getFinanceReceivables({ ...snapshot, limit: meta.value.limit, offset: meta.value.offset })
    if (disposed || id !== requestID) {
      return
    }
    rows.value = response.data || []
    summary.value = response.summary || {}
    agingRows.value = response.aging || []
    meta.value = { ...meta.value, ...response.meta }
    applied.value = snapshot
    updatedAt.value = new Date()
  } catch (error) {
    if (!disposed && id === requestID) {
      rows.value = []
      summary.value = {}
      agingRows.value = []
      meta.value.total = 0
      errorMessage.value = error.response?.data?.error || 'Piutang belum dapat dimuat. Pastikan API dan migrasi database sudah terpasang.'
    }
  } finally {
    if (!disposed && id === requestID) {
      loading.value = false
    }
  }
}

const changePage = direction => {
  const offset = meta.value.offset + direction * meta.value.limit
  if (offset >= 0 && offset < Number(meta.value.total)) {
    meta.value.offset = offset
    load()
  }
}

const viewInvoice = row => {
  try {
    sessionStorage.setItem(invoiceReturnStorageKey, JSON.stringify({
      filters: { ...applied.value }, offset: meta.value.offset,
      scrollY: window.scrollY, savedAt: Date.now(),
    }))
  } catch { /* Navigation still works if storage is unavailable. */ }
  selected.value = null
  router.push({ path: '/finance/sales', query: { invoice: row.id, from: 'receivables' } })
}

const resetFilters = () => {
  Object.assign(filters, { query: '', date_from: '', date_to: '', status: 'OPEN', aging: 'ALL' })
  load(true)
}

const collectReportRows = async snapshot => {
  const result = []
  let offset = 0
  let total = Infinity
  while (offset < total) {
    const response = await getFinanceReceivables({ ...snapshot, offset, limit: 1000 })
    const batch = response.data || []
    total = Number(response.meta?.total || 0)
    if (!batch.length) {
      break
    }
    result.push(...batch)
    offset += batch.length
  }
  return result
}

const escapeHTML = value => String(value ?? '').replace(/[&<>"']/g, char => ({
  '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;',
}[char]))

const reportFilterLabel = snapshot => [
  snapshot.query ? `Pencarian: ${snapshot.query}` : 'Semua customer',
  statuses.find(item => item[0] === snapshot.status)?.[1] || snapshot.status,
  snapshot.aging === 'ALL' ? 'Semua aging' : ageOptions.find(item => item[0] === snapshot.aging)?.[1],
  snapshot.date_from || snapshot.date_to ? `${snapshot.date_from || 'Awal data'} sampai ${snapshot.date_to || 'Sekarang'}` : 'Seluruh tanggal faktur',
].filter(Boolean).join(' · ')

const printReport = async () => {
  if (!canReport.value) {
    return
  }
  const popup = window.open('', '_blank')
  if (!popup) {
    errorMessage.value = 'Izinkan pop-up localhost untuk membuka dokumen cetak'
    return
  }
  popup.opener = null
  popup.document.write('<p>Menyiapkan rekap piutang…</p>')
  exporting.value = true
  const snapshot = { ...applied.value }
  const reportSummary = { ...summary.value }
  try {
    const allRows = await collectReportRows(snapshot)
    const body = allRows.map((row, index) => `<tr>
      <td>${index + 1}</td><td>${escapeHTML(row.invoice_number)}<small>${escapeHTML(row.order_number)}</small></td>
      <td>${escapeHTML(row.store_name)}<small>${escapeHTML(row.customer_code)}</small></td>
      <td>${escapeHTML(date(row.invoice_date))}</td><td>${escapeHTML(date(row.due_date))}</td>
      <td class="money">${escapeHTML(money(row.total))}</td><td class="money">${escapeHTML(money(row.paid_amount))}</td>
      <td class="money">${escapeHTML(money(row.outstanding))}</td><td>${escapeHTML(dueLabels[row.due_status])}</td>
      <td>${Math.max(0, Number(row.overdue_days || 0))} hari</td></tr>`).join('')
    popup.document.open()
    popup.onload = () => {
      popup.focus()
      popup.print()
    }
    popup.document.write(`<!doctype html><html lang="id"><head><meta charset="UTF-8"><title>Rekap Piutang PT FAS</title>
      <style>@page{size:A4 landscape;margin:10mm}body{font:11px Arial,sans-serif;color:#172033}h1{font-size:20px}p{line-height:1.5}table{width:100%;border-collapse:collapse}th,td{border:1px solid #cbd5e1;padding:6px;text-align:left}th{background:#f1f5f9}small{display:block;margin-top:3px;color:#475569}.money{text-align:right;white-space:nowrap}thead{display:table-header-group}tr{break-inside:avoid}footer{margin-top:14px;font-size:10px}.actions{margin-bottom:14px}@media print{.actions{display:none}}</style>
      </head><body>${companyDocumentHTML()}<div class="actions"><button onclick="window.print()">Cetak / Simpan PDF</button></div>
      <h1>PT FURI ANUGERAH SEJAHTERA — Rekap Piutang</h1><p>${escapeHTML(reportFilterLabel(snapshot))}<br>Posisi per ${escapeHTML(date(meta.value.as_of))}</p>
      <p>Faktur: ${allRows.length} · Sisa piutang: ${escapeHTML(money(reportSummary.outstanding))} · Terlambat: ${escapeHTML(money(reportSummary.overdue))}</p>
      <table><thead><tr><th>No</th><th>Faktur / SO</th><th>Customer</th><th>Tanggal</th><th>Jatuh tempo</th><th>Total</th><th>Dibayar tercatat</th><th>Sisa</th><th>Status tempo</th><th>Terlambat</th></tr></thead><tbody>${body}</tbody></table>
      <footer>Pembayaran dihitung dari catatan pembayaran yang dikonfirmasi di sistem. Dokumen mengikuti filter aktif.</footer></body></html>`)
    popup.document.close()
  } catch (error) {
    popup.close()
    errorMessage.value = error.response?.data?.error || 'Rekap piutang belum dapat dicetak'
  } finally {
    exporting.value = false
  }
}

const exportExcel = async () => {
  if (!canReport.value) {
    return
  }
  exporting.value = true
  try {
    const snapshot = { ...applied.value }
    const allRows = await collectReportRows(snapshot)
    downloadFinanceXlsx(`PTFAS_Piutang_${meta.value.as_of || 'rekap'}.xlsx`, [
      ['PT FAS — Rekap Piutang', reportFilterLabel(snapshot)],
      ['Faktur', 'Sales Order', 'Customer', 'Kode Customer', 'Telepon', 'NPWP/NIK', 'Gudang', 'Tanggal Faktur', 'Tempo Hari', 'Jatuh Tempo', 'Total', 'Dibayar Tercatat', 'Sisa Piutang', 'Status Pembayaran', 'Status Tempo', 'Hari Terlambat'],
      ...allRows.map(row => [row.invoice_number, row.order_number, row.store_name, row.customer_code, row.customer_phone, row.customer_tax_id_number, row.warehouse_name, row.invoice_date, row.payment_terms_days, row.due_date, Number(row.total), Number(row.paid_amount), Number(row.outstanding), row.payment_status, dueLabels[row.due_status], Math.max(0, Number(row.overdue_days || 0))]),
    ])
  } catch (error) {
    errorMessage.value = error.response?.data?.error || 'Data piutang belum dapat diekspor'
  } finally {
    exporting.value = false
  }
}

watch(() => filters.query, () => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => load(true), 350)
})
watch(() => [filters.status, filters.aging], () => load(true))
useModalDialog(() => selected.value ? '.receivable-dialog' : null, () => { selected.value = null })

onMounted(async () => {
  await load()
  if (disposed) return
  if (invoiceReturnState) {
    await nextTick()
    window.scrollTo({ top: Math.max(0, Number(invoiceReturnState.scrollY) || 0), behavior: 'auto' })
  }
  refreshTimer = setInterval(() => {
    if (!loading.value && !exporting.value) {
      load()
    }
  }, 15000)
})

onBeforeUnmount(() => {
  disposed = true
  requestID += 1
  clearInterval(refreshTimer)
  clearTimeout(searchTimer)
})
</script>

<template>
  <div class="receivables-page">
    <header class="page-header">
      <div>
        <p class="kicker">Finance · Receivables</p>
        <h1>Piutang & Jatuh Tempo</h1>
        <p class="description">Kontrol tagihan customer, pembayaran tercatat, dan umur piutang dalam satu halaman.</p>
      </div>
      <div class="header-actions">
        <button class="button secondary" :disabled="loading || exporting" @click="load()"><FasIcon name="refresh" /> Perbarui</button>
        <button class="button secondary" :disabled="!canReport" @click="exportExcel">Excel (.xlsx)</button>
        <button class="button primary" :disabled="!canReport" @click="printReport">{{ exporting ? 'Menyiapkan…' : 'Print / PDF' }}</button>
      </div>
    </header>
    <FinanceNavigation />
    <div v-if="errorMessage" class="error" role="alert">{{ errorMessage }}</div>
    <section class="summary-grid">
      <article><span>Sisa Piutang</span><strong>{{ money(summary.outstanding) }}</strong><small>{{ number(summary.invoice_count) }} faktur · {{ number(summary.customer_count) }} customer sesuai filter</small></article>
      <article><span>Terlambat</span><strong class="danger">{{ money(summary.overdue) }}</strong><small>{{ money(summary.over_30) }} melewati 30 hari</small></article>
      <article><span>Jatuh Tempo Hari Ini</span><strong class="warning">{{ money(summary.due_today) }}</strong><small>Posisi per {{ date(meta.as_of) }}</small></article>
      <article><span>Pembayaran Tercatat</span><strong>{{ money(summary.paid_amount) }}</strong><small>Total faktur {{ money(summary.total) }}</small></article>
    </section>
    <section class="panel aging-panel">
      <div class="section-heading"><div><h2>Aging Piutang</h2><p>Kelompok umur dihitung dari tanggal jatuh tempo, memakai tanggal Jakarta.</p></div><span>{{ periodLabel }}</span></div>
      <div class="aging-grid">
        <button v-for="item in aging" :key="item.key" :class="['aging-card', { active: filters.aging === item.key }]" @click="filters.aging = filters.aging === item.key ? 'ALL' : item.key">
          <span>{{ item.label }}</span><strong>{{ money(item.amount) }}</strong><small>{{ item.count }} faktur</small>
          <div class="meter"><span :style="{ width: Number(summary.outstanding) > 0 ? Math.min(100, item.amount / Number(summary.outstanding) * 100) + '%' : '0%' }"></span></div>
        </button>
      </div>
    </section>
    <section class="panel">
      <div class="filters">
        <label class="search-field"><span>Cari tagihan</span><div><FasIcon name="search" /><input v-model="filters.query" type="search" placeholder="Toko, faktur, SO, NPWP, telepon atau gudang" /></div></label>
        <label><span>Status</span><FasSelect v-model="filters.status" aria-label="Status piutang"><option v-for="[key, label] in statuses" :key="key" :value="key">{{ label }}</option></FasSelect></label>
        <label><span>Aging</span><FasSelect v-model="filters.aging" aria-label="Aging piutang"><option value="ALL">Semua aging</option><option v-for="[key, label] in ageOptions" :key="key" :value="key">{{ label }}</option></FasSelect></label>
        <label><span>Tanggal faktur dari</span><input v-model="filters.date_from" type="date" /></label>
        <label><span>Sampai</span><input v-model="filters.date_to" type="date" /></label>
        <div class="filter-actions"><button class="button primary" :disabled="loading" @click="load(true)">Terapkan</button><button class="button secondary" :disabled="loading" @click="resetFilters">Reset</button></div>
      </div>
      <div class="table-meta"><span>{{ periodLabel }} · {{ number(meta.total) }} tagihan sesuai filter</span><span>{{ loading ? 'Memuat…' : updatedAt ? 'Diperbarui ' + updatedAt.toLocaleTimeString('id-ID') : '' }}</span></div>
      <div class="table-scroll" :aria-busy="loading">
        <table><thead><tr><th>Faktur / Customer</th><th>Tanggal</th><th>Jatuh Tempo</th><th class="money">Total Faktur</th><th class="money">Dibayar Tercatat</th><th class="money">Sisa Piutang</th><th>Status</th><th>Aksi</th></tr></thead>
          <tbody><tr v-for="row in rows" :key="row.id">
            <td><button class="invoice-link" @click="viewInvoice(row)" :aria-label="`Lihat faktur ${row.invoice_number}`">{{ row.invoice_number }}</button><span>{{ row.store_name }}</span><small>{{ row.customer_code }} · {{ row.order_number }}</small></td>
            <td>{{ date(row.invoice_date) }}<small>Tempo {{ row.payment_terms_days }} hari</small></td>
            <td>{{ date(row.due_date) }}<small v-if="Number(row.overdue_days) > 0" class="danger">Terlambat {{ row.overdue_days }} hari</small></td>
            <td class="money">{{ money(row.total) }}</td><td class="money">{{ money(row.paid_amount) }}</td><td class="money outstanding">{{ money(row.outstanding) }}</td>
            <td><span :class="['badge', row.due_status.toLowerCase()]">{{ dueLabels[row.due_status] }}</span><small v-if="row.payment_status === 'PARTIAL'">Dibayar sebagian</small></td>
            <td><div class="receivable-actions"><button class="detail-button" @click="viewInvoice(row)">Lihat Faktur</button><button class="detail-button" @click="selected = row">Detail Piutang</button><button class="detail-button" @click="router.push({ path: '/finance/payments', query: { invoice: row.id } })">Pembayaran</button></div></td>
          </tr><tr v-if="!rows.length"><td colspan="8" class="empty">{{ loading ? 'Memuat tagihan…' : 'Tidak ada faktur tempo yang sesuai filter.' }}</td></tr></tbody>
        </table>
      </div>
      <footer class="table-footer"><span>Halaman {{ currentPage }} dari {{ pages }}</span><div><button class="button secondary" :disabled="loading || currentPage <= 1" @click="changePage(-1)">Sebelumnya</button><button class="button secondary" :disabled="loading || currentPage >= pages" @click="changePage(1)">Berikutnya</button></div></footer>
    </section>
    <p class="record-note">Angka pembayaran berasal dari catatan yang dikonfirmasi di sistem. Pembayaran historis yang belum dicatat perlu direkonsiliasi melalui modul Pembayaran.</p>
    <Teleport to="body"><div v-if="selected" class="receivable-modal-shell"><button class="modal-backdrop" aria-label="Tutup detail piutang" @click="selected = null"></button><section class="receivable-dialog" role="dialog" aria-modal="true" aria-label="Detail piutang">
      <header><div><p class="kicker">Detail Tagihan</p><h2>{{ selected.invoice_number }}</h2><span>{{ selected.store_name }}</span></div><button class="close-button" aria-label="Tutup detail" @click="selected = null"><FasIcon name="close" /></button></header>
      <div class="dialog-body"><div class="detail-grid"><div><span>Customer</span><strong>{{ selected.store_name }}</strong><small>{{ selected.customer_code }}</small></div><div><span>Kontak</span><strong>{{ selected.customer_phone || 'Belum diisi' }}</strong><small>NPWP/NIK: {{ selected.customer_tax_id_number || 'Belum diisi' }}</small></div><div><span>Sales Order</span><strong>{{ selected.order_number }}</strong><small>{{ selected.warehouse_name || 'Gudang belum diatur' }}</small></div><div><span>Jatuh Tempo</span><strong>{{ date(selected.due_date) }}</strong><small>{{ dueLabels[selected.due_status] }}</small></div><div class="wide"><span>Alamat Pengiriman</span><strong>{{ selected.shipping_address || 'Belum diisi' }}</strong></div></div>
      <div class="balance"><div><span>Total Faktur</span><strong>{{ money(selected.total) }}</strong></div><div><span>Pembayaran Tercatat</span><strong>{{ money(selected.paid_amount) }}</strong></div><div><span>Sisa Piutang</span><strong>{{ money(selected.outstanding) }}</strong></div></div>
      <p v-if="Number(selected.outstanding) > 0 && Number(selected.overdue_days) > 30" class="collection-note">Tagihan melewati 30 hari. Finance dapat menyiapkan tindak lanjut penagihan bersama pihak lapangan sesuai prosedur perusahaan.</p></div>
      <footer><button class="button secondary" @click="selected = null">Tutup</button><button class="button primary" @click="viewInvoice(selected)">Lihat Faktur</button><button class="button secondary" @click="router.push({ path: '/finance/payments', query: { invoice: selected.id } })">Catat Pembayaran</button></footer>
    </section></div></Teleport>
  </div>
</template>

<style scoped>
.receivable-actions { display: flex; flex-wrap: wrap; gap: .5rem; }

.invoice-link { color: var(--fas-accent-text); text-align: left; font: inherit; font-weight: 700; cursor: pointer; text-decoration: underline; text-underline-offset: 3px; }
.receivables-page { padding: 1.5rem; color: var(--fas-text); min-width: 0; }
.page-header { display: flex; align-items: flex-end; justify-content: space-between; flex-wrap: wrap; gap: 1rem; }
.kicker { color: var(--fas-accent-text); font-size: 0.68rem; letter-spacing: 0.1em; text-transform: uppercase; font-weight: 800; }
h1 { font-size: 1.8rem; font-weight: 800; letter-spacing: -0.03em; margin-top: 0.4rem; }
.description { color: var(--fas-muted); font-size: 0.85rem; margin-top: 0.5rem; line-height: 1.6; }
.header-actions, .filter-actions { display: flex; gap: 0.55rem; flex-wrap: wrap; }
.button { display: inline-flex; min-height: 2.85rem; align-items: center; justify-content: center; gap: 0.45rem; border: 1px solid var(--fas-border-strong); border-radius: 0.85rem; padding: 0.6rem 0.85rem; font-size: 0.8rem; font-weight: 700; cursor: pointer; }
.primary { background: var(--fas-accent); color: #ffffff; border-color: var(--fas-accent); }
.secondary { background: var(--fas-card); color: var(--fas-text-soft); }
button:disabled { opacity: 0.5; cursor: not-allowed; }
button:focus-visible { outline: 2px solid var(--fas-accent); outline-offset: 2px; }
.finance-nav { display: flex; gap: 0.4rem; flex-wrap: wrap; margin: 1.2rem 0; }
.finance-nav button { border: 1px solid var(--fas-border); background: var(--fas-card); color: var(--fas-muted); border-radius: 0.75rem; padding: 0.55rem 0.75rem; font-size: 0.75rem; cursor: pointer; }
.finance-nav .active, .aging-card.active { border-color: var(--fas-accent-border); background: var(--fas-accent-soft); color: var(--fas-accent-text); }
.error { margin-bottom: 1rem; border: 1px solid var(--fas-danger); background: var(--fas-danger-soft); color: var(--fas-danger); border-radius: 0.85rem; padding: 0.8rem; font-size: 0.8rem; }
.summary-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.85rem; }
.summary-grid article, .panel { border: 1px solid var(--fas-border); border-radius: 1.1rem; background: var(--fas-card); box-shadow: var(--fas-shadow-soft); }
.summary-grid article { padding: 1.1rem; }
.summary-grid span, .summary-grid small { display: block; color: var(--fas-muted); font-size: 0.75rem; }
.summary-grid strong { display: block; margin: 0.6rem 0; font-size: 1.3rem; overflow-wrap: anywhere; }
.summary-grid small { line-height: 1.5; font-size: 0.68rem; }
.danger { color: var(--fas-danger) !important; }
.warning { color: var(--fas-warning) !important; }
.panel { margin-top: 1rem; min-width: 0; overflow: hidden; }
.aging-panel { padding: 1rem; }
.section-heading { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; margin-bottom: 0.9rem; }
h2 { font-size: 1rem; font-weight: 800; }
.section-heading p, .section-heading > span { font-size: 0.72rem; color: var(--fas-muted); line-height: 1.5; margin-top: 0.3rem; }
.aging-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 0.65rem; }
.aging-card { text-align: left; border: 1px solid var(--fas-border); background: var(--fas-elevated); border-radius: 0.85rem; padding: 0.85rem; cursor: pointer; }
.aging-card > span, .aging-card small { display: block; font-size: 0.7rem; color: var(--fas-muted); }
.aging-card strong { display: block; margin: 0.4rem 0; font-size: 0.85rem; color: var(--fas-text); }
.meter { height: 4px; background: var(--fas-border); border-radius: 999px; margin-top: 0.7rem; overflow: hidden; }
.meter span { display: block; height: 100%; background: var(--fas-accent); border-radius: inherit; }
.filters { display: grid; grid-template-columns: minmax(260px, 2fr) minmax(150px, 1fr) minmax(170px, 1fr); gap: 0.9rem; padding: 1.1rem; }
.filters label { min-width: 0; }
.filters label > span { display: block; font-size: 0.72rem; font-weight: 700; margin-bottom: 0.4rem; color: var(--fas-text-soft); }
.filters input { min-height: 2.85rem; width: 100%; min-width: 0; border: 1px solid var(--fas-border-strong); border-radius: 0.85rem; background: var(--fas-input); padding: 0.6rem 0.8rem; color: var(--fas-text); font-size: 0.8rem; outline: none; }
.filters input:focus { border-color: var(--fas-accent-border); box-shadow: 0 0 0 3px var(--fas-accent-soft); }
.filters input::placeholder { color: var(--fas-muted); }
.search-field > div { position: relative; }
.search-field .fas-icon { position: absolute; left: 0.8rem; top: 0.85rem; color: var(--fas-muted); }
.search-field input { padding-left: 2.5rem; }
.filter-actions { align-items: flex-end; }
.table-meta { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 0.5rem; padding: 0.8rem 1.1rem; border-top: 1px solid var(--fas-border); font-size: 0.7rem; color: var(--fas-muted); }
.table-scroll { overflow-x: auto; scrollbar-width: thin; scrollbar-color: var(--fas-muted-soft) transparent; }
table { width: 100%; min-width: 1100px; border-collapse: collapse; font-size: 0.78rem; }
th { background: var(--fas-table-head); color: var(--fas-muted); text-align: left; font-size: 0.68rem; }
th, td { padding: 0.9rem 1rem; border-top: 1px solid var(--fas-border); vertical-align: middle; }
td > strong, td > span, td small { display: block; }
td > span { margin-top: 0.3rem; }
td small { margin-top: 0.35rem; color: var(--fas-muted); font-size: 0.66rem; }
tbody tr:hover { background: var(--fas-table-hover); }
.money { text-align: right; white-space: nowrap; }
.outstanding { color: var(--fas-accent-text); font-weight: 800; }
.badge { display: inline-flex; width: fit-content; border-radius: 999px; padding: 0.3rem 0.55rem; font-size: 0.65rem; background: var(--fas-overlay); color: var(--fas-muted); }
.overdue { background: var(--fas-danger-soft); color: var(--fas-danger); }
.due { background: var(--fas-warning-soft); color: var(--fas-warning); }
.paid { background: var(--fas-success-soft); color: var(--fas-success); }
.not_due { background: var(--fas-accent-soft); color: var(--fas-accent-text); }
.detail-button { border: 1px solid var(--fas-accent-border); border-radius: 0.65rem; background: var(--fas-accent-soft); color: var(--fas-accent-text); padding: 0.45rem 0.65rem; font-size: 0.72rem; cursor: pointer; }
.empty { text-align: center; color: var(--fas-muted); padding: 3rem; }
.table-footer { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 0.7rem; padding: 0.9rem 1rem; border-top: 1px solid var(--fas-border); font-size: 0.72rem; color: var(--fas-muted); }
.table-footer > div { display: flex; gap: 0.5rem; }
.record-note { color: var(--fas-muted); font-size: 0.72rem; margin-top: 0.9rem; line-height: 1.6; }
.receivable-modal-shell { position: fixed; z-index: 1300; inset: 0; padding: 1rem; display: grid; place-items: center; }
.modal-backdrop { position: absolute; inset: 0; border: 0; background: rgba(2, 6, 23, 0.55); backdrop-filter: blur(4px); }
.receivable-dialog { position: relative; display: flex; flex-direction: column; width: min(54rem, 100%); max-height: calc(100dvh - 2rem); border: 1px solid var(--fas-border-strong); border-radius: 1.15rem; background: var(--fas-card); color: var(--fas-text); box-shadow: var(--fas-shadow-card); overflow: hidden; }
.receivable-dialog header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; padding: 1.2rem; border-bottom: 1px solid var(--fas-border); flex-shrink: 0; }
.receivable-dialog header h2 { margin: 0.4rem 0; font-size: 1.2rem; }
.receivable-dialog header span { color: var(--fas-muted); font-size: 0.8rem; }
.close-button { display: grid; place-items: center; width: 2.5rem; height: 2.5rem; border: 1px solid var(--fas-border); border-radius: 0.75rem; background: var(--fas-elevated); color: var(--fas-text-soft); cursor: pointer; }
.dialog-body { padding: 1.2rem; overflow-y: auto; min-height: 0; }
.detail-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.detail-grid .wide { grid-column: span 2; }
.detail-grid span, .balance span { display: block; color: var(--fas-muted); font-size: 0.72rem; }
.detail-grid strong { display: block; font-size: 0.85rem; margin-top: 0.35rem; overflow-wrap: anywhere; }
.detail-grid small { display: block; color: var(--fas-muted); font-size: 0.68rem; margin-top: 0.3rem; }
.balance { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 0.8rem; margin-top: 1.2rem; background: var(--fas-accent-soft); border: 1px solid var(--fas-accent-border); border-radius: 0.9rem; padding: 1rem; }
.balance strong { display: block; margin-top: 0.4rem; font-size: 1rem; overflow-wrap: anywhere; }
.collection-note { padding: 0.8rem; background: var(--fas-warning-soft); color: var(--fas-warning); border-radius: 0.75rem; margin-top: 1rem; font-size: 0.8rem; line-height: 1.5; }
.receivable-dialog footer { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 0.6rem; padding: 0.9rem 1.2rem; border-top: 1px solid var(--fas-border); flex-shrink: 0; }
@media (max-width: 1100px) { .summary-grid, .aging-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); } .filters { grid-template-columns: repeat(2, minmax(0, 1fr)); } .search-field { grid-column: span 2; } }
@media (max-width: 640px) { .receivables-page { padding: 1rem; } .summary-grid, .aging-grid, .filters, .detail-grid, .balance { grid-template-columns: 1fr; } .search-field, .detail-grid .wide { grid-column: auto; } .header-actions { width: 100%; } .header-actions .button { flex: 1; } h1 { font-size: 1.55rem; } .receivable-dialog footer .button { flex: 1; } }
</style>
