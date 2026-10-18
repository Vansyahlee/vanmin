<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FasSelect from '@/components/ui/FasSelect.vue'
import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import InvoicePaymentPanel from '@/components/finance/InvoicePaymentPanel.vue'
import { getFinancePayments, getFinanceReceivables, getFinancePaymentConfig, testFinancePaymentConnection } from '@/services/finance'
import { downloadFinanceXlsx, openFinancePrint, writeFinancePrint } from '@/utils/financeDocuments'
const route = useRoute()
const router = useRouter()
const connectionBusy = ref(false)
const connectionError = ref('')
const connectionMessage = ref('')
const gatewayConfig = ref({ enabled: false, environment: 'sandbox' })
const checkConnection = async () => {
  connectionBusy.value = true; connectionError.value = ''; connectionMessage.value = ''
  try { const response = await testFinancePaymentConnection(); connectionMessage.value = response.message }
  catch (failure) { connectionError.value = failure.response?.data?.error || 'Midtrans belum dapat dihubungi.' }
  finally { connectionBusy.value = false }
}
getFinancePaymentConfig().then(response => { gatewayConfig.value = response }).catch(() => {})
const panel = ref(null)
const invoiceID = computed(() => typeof route.query.invoice === 'string' && /^[0-9a-f-]{36}$/i.test(route.query.invoice) ? route.query.invoice : '')
const invoiceQuery = ref('')
const invoiceOptions = ref([])
const searching = ref(false)
const pickerError = ref('')
const rows = ref([])
const summary = ref({})
const meta = ref({ total: 0, offset: 0, limit: 50 })
const loading = ref(false)
const exporting = ref(false)
const error = ref('')
const filters = reactive({ query: '', date_from: '', date_to: '', status: 'ALL' })
const applied = ref({ ...filters })
let listSequence = 0, searchSequence = 0, disposed = false, searchTimer, invoiceTimer, refreshTimer
const navigation = [['overview', 'Overview', '/finance'], ['analytics', 'Analytics', '/finance/analytics'], ['sales', 'Penjualan', '/finance/sales'], ['customers', 'Customer', '/finance/customers'], ['receivables', 'Piutang', '/finance/receivables'], ['payments', 'Pembayaran', '/finance/payments'], ['expenses', 'Pengeluaran', '/finance/expenses'], ['cash', 'Kas & Bank', '/finance/cash'], ['profit', 'Laba', '/finance/profit'], ['tax', 'Pajak', '/finance/tax'], ['reports', 'Laporan', '/finance/reports']]
const money = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 2 }).format(Number(value || 0))
const date = value => new Date(value).toLocaleString('id-ID', { timeZone: 'Asia/Jakarta' })
const page = computed(() => Math.floor(meta.value.offset / meta.value.limit) + 1)
const pages = computed(() => Math.max(1, Math.ceil(Number(meta.value.total) / meta.value.limit)))
const ready = computed(() => !loading.value && !exporting.value && !error.value && Number(meta.value.total) > 0)
const searchInvoices = async () => {
  const id = ++searchSequence
  searching.value = true; pickerError.value = ''
  try {
    const response = await getFinanceReceivables({ query: invoiceQuery.value, status: 'ALL', aging: 'ALL', limit: 50, offset: 0 })
    if (!disposed && id === searchSequence) invoiceOptions.value = response.data || []
  } catch (failure) {
    if (!disposed && id === searchSequence) pickerError.value = failure.response?.data?.error || 'Pilihan faktur belum dapat dimuat.'
  } finally { if (!disposed && id === searchSequence) searching.value = false }
}
const selectInvoice = async value => {
  await router.replace({ path: '/finance/payments', query: value ? { invoice: value } : {} })
  await nextTick(); if (value) panel.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
const load = async (reset = false) => {
  if (filters.date_from && filters.date_to && filters.date_from > filters.date_to) { error.value = 'Tanggal awal melebihi tanggal akhir.'; return }
  if (reset) meta.value.offset = 0
  const id = ++listSequence, snapshot = { ...filters }
  loading.value = true; error.value = ''
  try {
    const result = await getFinancePayments({ ...snapshot, limit: meta.value.limit, offset: meta.value.offset })
    if (disposed || id !== listSequence) return
    rows.value = result.data || []; summary.value = result.summary || {}; meta.value = { ...meta.value, ...result.meta }; applied.value = snapshot
  } catch (failure) {
    if (!disposed && id === listSequence) { error.value = failure.response?.data?.error || 'Pembayaran belum dapat dimuat.'; rows.value = []; summary.value = {}; meta.value.total = 0 }
  } finally { if (!disposed && id === listSequence) loading.value = false }
}
const changed = () => { load(); searchInvoices() }
const reset = () => { Object.assign(filters, { query: '', date_from: '', date_to: '', status: 'ALL' }); load(true) }
const changePage = direction => {
  const next = meta.value.offset + direction * meta.value.limit
  if (next >= 0 && next < Number(meta.value.total)) { meta.value.offset = next; load() }
}
const collect = async snapshot => {
  const result = []
  let offset = 0, total = Infinity
  while (offset < total) {
    const response = await getFinancePayments({ ...snapshot, limit: 200, offset })
    const batch = response.data || []
    total = Number(response.meta?.total || 0)
    if (!batch.length) break
    result.push(...batch); offset += batch.length
  }
  return result
}
const headings = ['Nomor Pembayaran', 'Tanggal Bayar', 'Faktur', 'Customer', 'Metode', 'Sumber', 'Nominal', 'Refund', 'Pembayaran Setelah Refund', 'Status', 'Referensi', 'Catatan / Alasan Batal']
const reportRows = items => items.map(item => [item.payment_number, date(item.paid_at), item.invoice_number, item.store_name, item.method, item.source, Number(item.amount), Number(item.refunded_amount), Number(item.net_amount), item.status, item.reference_number, item.void_reason || item.notes])
const filterDescription = snapshot => `Tanggal pembayaran ${snapshot.date_from || 'awal data'} sampai ${snapshot.date_to || 'seluruh tanggal'} · Status ${snapshot.status} · Pencarian ${snapshot.query || 'semua'} · Waktu cetak ${new Date().toLocaleString('id-ID')}`
const report = async kind => {
  if (!ready.value) return
  let popup
  const snapshot = { ...applied.value }, net = summary.value.net_amount
  exporting.value = true
  try {
    if (kind === 'print') popup = openFinancePrint()
    const items = await collect(snapshot)
    const description = filterDescription(snapshot) + ` · Pembayaran setelah refund ${money(net)}`
    if (kind === 'print') writeFinancePrint(popup, 'PT FAS — Rekap Pembayaran', description, headings, reportRows(items))
    else downloadFinanceXlsx('PTFAS_Rekap_Pembayaran.xlsx', [['PT FAS — Rekap Pembayaran'], [description], headings, ...reportRows(items)])
  } catch (failure) { popup?.close(); error.value = failure.response?.data?.error || failure.message || 'Dokumen belum dapat dibuat.' }
  finally { exporting.value = false }
}
watch(invoiceQuery, () => { clearTimeout(invoiceTimer); invoiceTimer = setTimeout(searchInvoices, 350) })
watch(() => filters.query, () => { clearTimeout(searchTimer); searchTimer = setTimeout(() => load(true), 350) })
watch(() => filters.status, () => load(true))
onMounted(() => { load(true); searchInvoices(); refreshTimer = setInterval(() => { if (!loading.value && !exporting.value) load() }, 15000) })
onBeforeUnmount(() => { disposed = true; listSequence++; searchSequence++; clearTimeout(searchTimer); clearTimeout(invoiceTimer); clearInterval(refreshTimer) })
</script>

<template>
  <div class="finance-view payments-page">
    <header class="page-header"><div><p class="kicker">Finance · Payments</p><h1>Pembayaran Customer</h1><p class="description">Catat cicilan, pantau pelunasan, dan hubungkan pembayaran Midtrans ke faktur yang sama.</p></div><div class="actions"><button :disabled="loading" @click="load()">Perbarui</button><button :disabled="!ready" @click="report('excel')">Excel</button><button class="primary" :disabled="!ready" @click="report('print')">Print / PDF</button></div></header>
    <FinanceNavigation />
    <section class="panel picker"><div class="selected-header"><div><h2>Midtrans {{ gatewayConfig.environment === 'production' ? 'Production' : 'Sandbox' }}</h2><p class="muted">{{ gatewayConfig.enabled ? 'Konfigurasi backend aktif. Tes koneksi sebelum mencoba pembayaran.' : 'Belum aktif. Isi konfigurasi Midtrans pada .env backend lalu restart server.' }}</p></div><div class="actions"><button :disabled="connectionBusy" @click="checkConnection">{{ connectionBusy ? 'Menguji…' : 'Tes Koneksi Midtrans' }}</button></div></div><p v-if="connectionMessage" class="muted" role="status">{{ connectionMessage }}</p><p v-if="connectionError" class="error" role="alert">{{ connectionError }}</p></section>
    <section v-if="gatewayConfig.environment === 'sandbox'" class="panel picker">
      <h2>Cara Tes Pembayaran Sandbox</h2>
      <ol class="sandbox-steps">
        <li>Pilih faktur tempo yang masih memiliki sisa piutang. Faktur LUNAS / sisa Rp0 tidak bisa dibayar lagi. Lalu klik <strong>Catat Pembayaran / Cicilan</strong>.</li>
        <li>Isi nominal tanpa titik ribuan, misalnya <strong>100000</strong> untuk Rp100.000. Pilih Midtrans, lalu klik <strong>Buat Tautan Midtrans</strong> dan <strong>Buka Midtrans</strong>.</li>
        <li>Di checkout Sandbox pilih bank yang tersedia (contoh BCA atau BSI), lalu salin nomor VA dummy yang muncul. Nomor VA berasal dari checkout, bukan Merchant ID, rekening perusahaan, atau nomor faktur.</li>
        <li>Buka simulator untuk BANK YANG SAMA: <a href="https://simulator.sandbox.midtrans.com/bca/va/index" target="_blank" rel="noopener noreferrer">BCA</a> atau <a href="https://simulator.sandbox.midtrans.com/openapi/va/index?bank=bsi" target="_blank" rel="noopener noreferrer">BSI</a>. Tempel nomor VA dummy ke Virtual Account Number, klik Inquire, periksa nominal, lalu konfirmasi pembayaran sesuai simulator.</li>
        <li>Kembali ke halaman ini, pada tautan Midtrans klik <strong>Cek Status</strong>. Setelah terkonfirmasi, saldo piutang dan riwayat diperbarui. Contoh sisa Rp750.000 dibayar Rp100.000 → sisa Rp650.000.</li>
      </ol>
      <p class="muted">Gunakan simulator, bukan transfer bank/scan QR dengan uang sungguhan. Bila BCA tidak tersedia, ikuti <a href="https://docs.midtrans.com/docs/testing-payment-on-sandbox" target="_blank" rel="noopener noreferrer">panduan simulator untuk metode yang tersedia</a>. Notifikasi otomatis memerlukan URL backend publik; selama localhost gunakan Cek Status.</p>
    </section>
    <section class="panel picker"><div><h2>Pilih Faktur untuk Pembayaran</h2><p class="muted">Cari toko atau nomor faktur. Cicilan dan pelunasan selalu dicatat pada faktur yang dipilih.</p></div><div class="picker-grid"><label><span>Cari faktur / toko</span><input v-model="invoiceQuery" type="search" placeholder="Nama toko, nomor faktur, NPWP/NIK" /></label><label><span>{{ searching ? 'Memuat pilihan…' : 'Faktur tempo' }}</span><FasSelect :model-value="invoiceID" aria-label="Pilih faktur pembayaran" @update:model-value="selectInvoice"><option value="">Pilih faktur</option><option v-if="invoiceID && !invoiceOptions.some(item => item.id === invoiceID)" :value="invoiceID">Faktur yang dipilih dari Piutang</option><option v-for="item in invoiceOptions" :key="item.id" :value="item.id">{{ item.invoice_number }} · {{ item.store_name }} · Sisa {{ money(item.outstanding) }}</option></FasSelect></label></div><p v-if="pickerError" class="error">{{ pickerError }}</p><p class="muted">Pilihan menampilkan maksimal 50 hasil; ketik nomor atau nama untuk mempersempit pencarian.</p></section>
    <section v-if="invoiceID" ref="panel" class="panel selected-panel"><div class="selected-header"><h2>Pembayaran Faktur Terpilih</h2><button @click="selectInvoice('')">Tutup Detail</button></div><InvoicePaymentPanel :key="invoiceID" :invoice-id="invoiceID" @changed="changed" /></section>
    <section class="panel"><div class="ledger-heading"><div><h2>Rekap Penerimaan</h2><p class="muted">{{ meta.total }} catatan sesuai filter · Pembayaran setelah refund <strong>{{ money(summary.net_amount) }}</strong></p></div></div><div class="filters"><label><span>Pencarian</span><input v-model="filters.query" type="search" placeholder="Pembayaran, faktur, toko, referensi" /></label><label><span>Status catatan</span><FasSelect v-model="filters.status" aria-label="Status pembayaran"><option value="ALL">Semua catatan</option><option value="CONFIRMED">Terkonfirmasi</option><option value="VOID">Dibatalkan</option></FasSelect></label><label><span>Tanggal pembayaran dari</span><input v-model="filters.date_from" type="date" /></label><label><span>Sampai</span><input v-model="filters.date_to" type="date" /></label><div class="actions"><button class="primary" :disabled="loading || exporting" @click="load(true)">Terapkan</button><button :disabled="loading || exporting" @click="reset">Reset</button></div></div>
      <p v-if="error" class="error" role="alert">{{ error }}</p>
      <div class="table-scroll" :aria-busy="loading"><table><thead><tr><th>Tanggal / Pembayaran</th><th>Faktur / Customer</th><th>Metode</th><th class="money">Nominal</th><th class="money">Pembayaran Setelah Refund</th><th>Status</th><th>Aksi</th></tr></thead><tbody><tr v-for="item in rows" :key="item.id"><td>{{ date(item.paid_at) }}<small>{{ item.payment_number }}</small></td><td><strong>{{ item.invoice_number }}</strong><small>{{ item.store_name }}</small></td><td>{{ item.method }}<small>{{ item.source }} · {{ item.reference_number || 'Tanpa referensi' }}</small></td><td class="money">{{ money(item.amount) }}<small v-if="Number(item.refunded_amount)">Refund {{ money(item.refunded_amount) }}</small></td><td class="money">{{ money(item.net_amount) }}</td><td>{{ item.status === 'VOID' ? 'Dibatalkan / dikembalikan' : 'Terkonfirmasi' }}</td><td><div class="actions"><button @click="router.push({ path: '/finance/sales', query: { invoice: item.invoice_id, from: 'payments' } })">Lihat Faktur</button><button @click="selectInvoice(item.invoice_id)">Riwayat / Bukti</button></div></td></tr><tr v-if="!rows.length"><td colspan="7" class="empty">{{ loading ? 'Memuat pembayaran…' : error ? 'Data belum dapat dimuat.' : 'Belum ada pembayaran sesuai filter.' }}</td></tr></tbody></table></div>
      <footer><span>Halaman {{ page }} dari {{ pages }}</span><div class="actions"><button :disabled="loading || page <= 1" @click="changePage(-1)">Sebelumnya</button><button :disabled="loading || page >= pages" @click="changePage(1)">Berikutnya</button></div></footer>
    </section>
  </div>
</template>

<style scoped>
.payments-page{padding:1.5rem;color:var(--fas-text);min-width:0}.page-header,.actions,.selected-header{display:flex;gap:.6rem;flex-wrap:wrap;align-items:center}.page-header,.selected-header{justify-content:space-between}.kicker{color:var(--fas-accent-text);font-size:.68rem;font-weight:800;letter-spacing:.1em;text-transform:uppercase}h1{font-size:1.8rem;font-weight:800;margin:.4rem 0}h2{font-size:1rem;font-weight:800}.description,.muted{font-size:.8rem;color:var(--fas-muted);line-height:1.6;margin-top:.45rem}.actions button,.selected-header button{min-height:2.85rem;border:1px solid var(--fas-border-strong);background:var(--fas-card);color:var(--fas-text-soft);border-radius:.85rem;padding:.6rem .85rem;font-size:.8rem;font-weight:700;cursor:pointer}.actions .primary{background:var(--fas-accent);border-color:var(--fas-accent);color:white}.finance-nav{display:flex;flex-wrap:wrap;gap:.4rem;margin:1.2rem 0}.finance-nav button{min-height:2.4rem;border:1px solid var(--fas-border);border-radius:.75rem;padding:.55rem .75rem;font-size:.75rem;background:var(--fas-card);color:var(--fas-muted);cursor:pointer}.finance-nav .active{background:var(--fas-accent-soft);border-color:var(--fas-accent-border);color:var(--fas-accent-text)}.panel{border:1px solid var(--fas-border);border-radius:1.1rem;background:var(--fas-card);margin-top:1rem;overflow:hidden;min-width:0}.picker,.selected-panel{padding:1.1rem}.selected-panel{scroll-margin-top:7rem}.selected-header{margin-bottom:1rem}.picker-grid{display:grid;grid-template-columns:1fr 1.5fr;gap:.85rem;margin:.9rem 0}.filters{padding:1rem;display:grid;grid-template-columns:1.5fr 1fr 1fr 1fr;gap:.85rem;align-items:end}.filters .actions{grid-column:1/-1}label{display:flex;flex-direction:column;gap:.4rem;min-width:0;font-size:.75rem;color:var(--fas-text-soft)}input{width:100%;min-height:2.85rem;padding:.65rem .8rem;border:1px solid var(--fas-border-strong);border-radius:.85rem;background:var(--fas-input);color:var(--fas-text);font-size:.8rem}.ledger-heading{padding:1.1rem;border-bottom:1px solid var(--fas-border)}.error{margin:1rem;padding:.8rem;background:var(--fas-danger-soft);color:var(--fas-danger);font-size:.8rem;border-radius:.8rem}.table-scroll{overflow:auto}table{width:100%;min-width:1000px;border-collapse:collapse;font-size:.78rem}th,td{padding:.85rem 1rem;border-top:1px solid var(--fas-border);text-align:left;vertical-align:top}th{font-size:.72rem;color:var(--fas-muted);background:var(--fas-elevated)}small{display:block;font-size:.68rem;line-height:1.5;color:var(--fas-muted);margin-top:.3rem;overflow-wrap:anywhere}.money{text-align:right;white-space:nowrap}.empty{text-align:center;padding:2rem;color:var(--fas-muted)}footer{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:.75rem;padding:1rem;font-size:.75rem;color:var(--fas-muted);border-top:1px solid var(--fas-border)}button:disabled{opacity:.5;cursor:not-allowed}button:focus-visible,input:focus-visible{outline:2px solid var(--fas-accent);outline-offset:2px}@media(max-width:900px){.filters{grid-template-columns:1fr 1fr}.picker-grid{grid-template-columns:1fr}}@media(max-width:600px){.filters{grid-template-columns:1fr}.payments-page{padding:1rem}}
.sandbox-steps{padding-left:1.25rem;font-size:.85rem;line-height:1.8}.sandbox-steps li{margin:.5rem 0}.sandbox-steps a,.payments-page .muted a{color:var(--fas-accent-text);text-decoration:underline}
</style>
