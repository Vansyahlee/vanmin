<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import FasSelect from '@/components/ui/FasSelect.vue'
import { getFinanceInvoicePayments, getFinancePaymentConfig, createFinancePayment, voidFinancePayment, syncFinancePayment, recoverFinancePayment, cancelFinancePayment } from '@/services/finance'
import { downloadFinanceXlsx, openFinancePrint, writeFinancePrint } from '@/utils/financeDocuments'
const props = defineProps({ invoiceId: { type: String, required: true } })
const emit = defineEmits(['changed'])
const router = useRouter()
const data = ref(null)
const loading = ref(false)
const busy = ref(false)
const error = ref('')
const message = ref('')
const config = ref({ enabled: false, environment: 'sandbox' })
const amountInput = ref(null)
const formOpen = ref(false)
const submitted = ref(null)
const voidID = ref('')
const voidReason = ref('')
const showAudit = ref(false)
const form = reactive({ amount: '', method: 'CASH', reference_number: '', paid_at: '', notes: '' })
let requestKey = ''
let sequence = 0
let disposed = false
const money = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 2 }).format(Number(value || 0))
const date = value => value ? new Date(value).toLocaleString('id-ID', { timeZone: 'Asia/Jakarta' }) : '—'
const invoice = computed(() => data.value?.invoice || {})
const payments = computed(() => data.value?.payments || [])
const intents = computed(() => data.value?.intents || [])
const events = computed(() => data.value?.events || [])
const balanceState = computed(() => Number(invoice.value.outstanding) <= 0 ? 'Lunas' : Number(invoice.value.paid_amount) > 0 ? 'Dibayar sebagian' : 'Belum dibayar')
const environmentLabel = computed(() => config.value.environment === 'production' ? 'Production' : 'Sandbox · simulasi')
const intentState = item => ({ CREATING: 'Tautan perlu dipulihkan', PENDING: 'Menunggu pembayaran', CONFIRMED: 'Berhasil', RELEASED: 'Sesi dibatalkan', REFUNDED: 'Dikembalikan', VOID: 'Dibatalkan' }[item.state] || item.state)
const methodLabel = method => ({ CASH: 'Tunai', BANK_TRANSFER: 'Transfer bank', LEGACY: 'Catatan sebelumnya', MIDTRANS: 'Midtrans' }[method] || method)
const activeIntent = item => ['CREATING', 'PENDING'].includes(item.state)
const newForm = () => {
  if (submitted.value) { formOpen.value = true; return }
  requestKey = crypto.randomUUID()
  const now = new Date()
  const local = new Date(now.getTime() - now.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
  Object.assign(form, { amount: '', method: 'CASH', reference_number: '', paid_at: local, notes: '' })
  error.value = ''; message.value = ''; formOpen.value = true
  nextTick(() => amountInput.value?.focus())
}
const load = async (silent = false) => {
  const id = ++sequence
  if (!silent) loading.value = true
  try {
    const response = await getFinanceInvoicePayments(props.invoiceId)
    if (disposed || id !== sequence) return
    data.value = response
    if (!silent) error.value = ''
  } catch (failure) {
    if (!disposed && id === sequence) error.value = failure.response?.data?.error || 'Saldo dan riwayat pembayaran belum dapat dimuat.'
  } finally { if (!disposed && id === sequence) loading.value = false }
}
watch(() => props.invoiceId, async () => {
  data.value = null; submitted.value = null; formOpen.value = false; voidID.value = ''; message.value = ''
  await load()
}, { immediate: true })
getFinancePaymentConfig().then(result => { if (!disposed) config.value = result }).catch(() => {})
const save = async () => {
  if (busy.value || loading.value) return
  error.value = ''; message.value = ''
  if (!submitted.value) {
    if (!/^\d+(\.\d{1,2})?$/.test(form.amount) || Number(form.amount) <= 0) { error.value = 'Isi nominal tanpa titik ribuan, contoh: 2000000'; return }
    if (Number(form.amount) > Number(invoice.value.available_amount)) { error.value = 'Nominal melebihi saldo yang tersedia. Periksa tautan Midtrans yang masih aktif.'; return }
    if (form.method === 'BANK_TRANSFER' && !form.reference_number.trim()) { error.value = 'Isi nomor referensi transfer.'; return }
    const paidAt = new Date(form.paid_at)
    if (form.method !== 'MIDTRANS' && !Number.isFinite(paidAt.getTime())) { error.value = 'Isi tanggal pembayaran.'; return }
    submitted.value = {
      invoice_id: props.invoiceId, request_key: requestKey, amount: form.amount,
      source: form.method === 'MIDTRANS' ? 'MIDTRANS' : 'MANUAL',
      method: form.method, reference_number: form.reference_number.trim(), notes: form.notes.trim(),
      paid_at: form.method === 'MIDTRANS' ? '' : paidAt.toISOString(),
    }
  }
  busy.value = true
  try {
    const response = await createFinancePayment(submitted.value)
    message.value = response.data?.source === 'MIDTRANS' ? 'Tautan Midtrans siap. Buka atau salin tautannya di bawah. Saldo berkurang setelah pembayaran terverifikasi.' : 'Pembayaran tercatat. Saldo dihitung ulang dari riwayat.'
    submitted.value = null; formOpen.value = false
    await load(true)
    emit('changed', data.value?.invoice); window.dispatchEvent(new Event('fas-finance-notifications-changed'))
  } catch (failure) {
    error.value = failure.response?.data?.error || 'Hasil penyimpanan belum dapat dipastikan. Gunakan Ulangi Permintaan agar tidak membuat pembayaran ganda.'
    if (failure.response?.data?.intent || [400, 404, 409, 503].includes(failure.response?.status)) submitted.value = null
    emit('changed', data.value?.invoice); window.dispatchEvent(new Event('fas-finance-notifications-changed'))
    await load(true)
  } finally { busy.value = false }
}
const act = async (operation, id) => {
  if (busy.value) return
  busy.value = true; error.value = ''; message.value = ''
  try {
    const response = await operation(id)
    message.value = response.message || 'Status diperbarui.'
    await load(true); emit('changed', data.value?.invoice); window.dispatchEvent(new Event('fas-finance-notifications-changed'))
  } catch (failure) {
    error.value = failure.response?.data?.error || 'Permintaan belum berhasil.'
    await load(true); emit('changed', data.value?.invoice); window.dispatchEvent(new Event('fas-finance-notifications-changed'))
  } finally { busy.value = false }
}
const voidPayment = async () => {
  if (voidReason.value.trim().length < 5) { error.value = 'Isi alasan pembatalan minimal 5 karakter.'; return }
  const id = voidID.value, reason = voidReason.value.trim()
  await act(() => voidFinancePayment(id, reason), id)
  if (!error.value) { voidID.value = ''; voidReason.value = '' }
}
const copyLink = async item => {
  try { await navigator.clipboard.writeText(item.redirect_url); message.value = 'Tautan disalin. Kirim ke customer sesuai kebutuhan.' }
  catch { error.value = 'Browser tidak mengizinkan salin otomatis. Buka tautan lalu salin alamatnya.' }
}
const safeLink = item => {
  try {
    const target = new URL(item.redirect_url)
    const host = item.environment === 'production' ? 'app.midtrans.com' : 'app.sandbox.midtrans.com'
    return target.protocol === 'https:' && target.host === host && !target.username && !target.password ? target.href : ''
  } catch { return '' }
}
const headings = ['Nomor Pembayaran', 'Tanggal', 'Metode', 'Sumber', 'Nominal', 'Refund', 'Pembayaran Setelah Refund', 'Status', 'Referensi', 'Petugas', 'Catatan / Alasan Batal']
const reportRows = items => items.map(item => [item.payment_number, date(item.paid_at), methodLabel(item.method), item.source, Number(item.amount), Number(item.refunded_amount || 0), item.status === 'CONFIRMED' ? Number(item.net_amount) : 0, item.status, item.reference_number, item.created_by_name || item.created_by || 'Data sebelumnya', item.void_reason || item.notes])
const description = () => `${invoice.value.invoice_number} · ${invoice.value.store_name} · Total ${money(invoice.value.total)} · Dibayar bersih ${money(invoice.value.paid_amount)} · Sisa ${money(invoice.value.outstanding)} · Saldo saat dicetak ${new Date().toLocaleString('id-ID')}`
const print = (item = null) => {
  let popup
  try { popup = openFinancePrint(); writeFinancePrint(popup, item ? 'PT FAS — Bukti Pembayaran' : 'PT FAS — Riwayat Pembayaran', description(), headings, reportRows(item ? [item] : payments.value)) }
  catch (failure) { popup?.close(); error.value = failure.message }
}
const excel = () => downloadFinanceXlsx(`PTFAS_Riwayat_${props.invoiceId}.xlsx`, [['PT FAS — Riwayat Pembayaran'], [description()], headings, ...reportRows(payments.value)])
const refreshTimer = setInterval(() => { if (!busy.value && !loading.value) load(true) }, 15000)
onBeforeUnmount(() => { disposed = true; sequence++; clearInterval(refreshTimer) })
</script>

<template>
  <section class="payment-panel" :aria-busy="loading">
    <p v-if="error" class="notice danger" role="alert">{{ error }}</p>
    <p v-if="message" class="notice success" role="status">{{ message }}</p>
    <p v-if="loading && !data" class="muted">Memuat saldo dan riwayat…</p>
    <template v-if="data?.invoice">
      <header class="section-heading"><div><h3>Saldo & Riwayat Pembayaran</h3><p>{{ balanceState }} · {{ invoice.invoice_number }}</p></div><div class="actions"><button type="button" :disabled="loading || busy" @click="load()">Perbarui</button><button type="button" @click="router.push({ path: '/finance/sales', query: { invoice: invoiceId, from: 'payments' } })">Lihat Faktur</button></div></header>
      <div class="balances"><article><span>Total Faktur</span><strong>{{ money(invoice.total) }}</strong></article><article><span>Dibayar Tercatat</span><strong>{{ money(invoice.paid_amount) }}</strong></article><article><span>Sisa Piutang</span><strong>{{ money(invoice.outstanding) }}</strong></article></div>
      <p v-if="Number(invoice.reserved_amount) > 0" class="notice">{{ money(invoice.reserved_amount) }} sedang dicadangkan pada tautan Midtrans. Tersedia untuk pembayaran baru: {{ money(invoice.available_amount) }}. Tautan pending belum mengurangi piutang.</p>
      <div class="actions tools"><button class="primary" :disabled="busy || loading || Number(invoice.available_amount) <= 0" @click="newForm">Catat Pembayaran / Cicilan</button><button :disabled="!payments.length" @click="print()">Print / PDF Riwayat</button><button :disabled="!payments.length" @click="excel">Excel Riwayat</button></div>
      <form v-if="formOpen" class="payment-form" @submit.prevent="save">
        <h4>{{ submitted ? 'Ulangi permintaan yang sama' : 'Catat Pembayaran' }}</h4>
        <p v-if="submitted" class="muted">Hasil permintaan sebelumnya belum pasti. Isi dikunci saat mengulang agar satu pembayaran tidak tercatat dua kali.</p>
        <fieldset :disabled="busy || !!submitted"><div class="form-grid">
          <label><span>Nominal cicilan (Rp)</span><input ref="amountInput" v-model="form.amount" type="text" inputmode="decimal" placeholder="Contoh: 2000000" required /></label>
          <label><span>Metode</span><FasSelect v-model="form.method" :disabled="busy || !!submitted" aria-label="Metode pembayaran"><option value="CASH">Tunai — sudah diterima</option><option value="BANK_TRANSFER">Transfer bank — sudah diterima</option><option v-if="config.enabled" value="MIDTRANS">Midtrans — {{ environmentLabel }}</option></FasSelect></label>
          <label v-if="form.method !== 'MIDTRANS'"><span>Tanggal & jam pembayaran</span><input v-model="form.paid_at" type="datetime-local" required /></label>
          <label v-if="form.method !== 'MIDTRANS'"><span>Referensi {{ form.method === 'BANK_TRANSFER' ? '(wajib)' : '(opsional)' }}</span><input v-model="form.reference_number" maxlength="150" :required="form.method === 'BANK_TRANSFER'" placeholder="Nomor transfer / kuitansi" /></label>
          <label class="wide"><span>Catatan</span><textarea v-model="form.notes" rows="2" maxlength="2000" placeholder="Contoh: cicilan pertama dari tagihan Rp8 juta"></textarea></label>
        </div></fieldset>
        <p v-if="form.method === 'MIDTRANS'" class="muted">{{ environmentLabel }} · Membuat tautan sebesar nominal cicilan, bukan seluruh faktur. Pending belum dianggap pembayaran.</p>
        <p v-else class="muted">Gunakan setelah uang benar-benar diterima. Total faktur tetap; pembayaran menambah riwayat dan mengurangi saldo.</p>
        <p v-if="!config.enabled" class="muted">Pembayaran tunai dan transfer siap digunakan. Midtrans tersedia setelah konfigurasi backend diaktifkan.</p>
        <div class="actions"><button type="button" :disabled="busy" @click="formOpen = false">Tutup</button><button class="primary" type="submit" :disabled="busy">{{ busy ? 'Memproses…' : submitted ? 'Ulangi Permintaan' : form.method === 'MIDTRANS' ? 'Buat Tautan Midtrans' : 'Simpan Pembayaran' }}</button></div>
      </form>
      <h4>Riwayat Penerimaan</h4>
      <div class="table-scroll"><table><thead><tr><th>Tanggal / Nomor</th><th>Metode / Petugas</th><th class="money">Nominal</th><th class="money">Refund</th><th>Status</th><th>Aksi</th></tr></thead><tbody><tr v-for="item in payments" :key="item.id"><td>{{ date(item.paid_at) }}<small>{{ item.payment_number }}</small><small>{{ item.reference_number || item.notes }}</small></td><td>{{ methodLabel(item.method) }}<small>{{ item.created_by_name || item.created_by || 'Data sebelumnya' }}</small></td><td class="money">{{ money(item.amount) }}</td><td class="money">{{ money(item.refunded_amount) }}</td><td><span class="badge">{{ item.status === 'VOID' ? 'Dibatalkan / dikembalikan' : 'Terkonfirmasi' }}</span><small v-if="item.void_reason">{{ item.void_reason }}</small></td><td><div class="actions"><button @click="print(item)">Bukti / PDF</button><button v-if="item.source === 'MANUAL' && item.status === 'CONFIRMED'" :disabled="busy" @click="voidID = item.id; voidReason = ''">Batalkan Catatan</button></div></td></tr><tr v-if="!payments.length"><td colspan="6" class="empty">Belum ada pembayaran tercatat.</td></tr></tbody></table></div>
      <form v-if="voidID" class="payment-form" @submit.prevent="voidPayment"><h4>Batalkan Catatan Pembayaran Manual</h4><p class="muted">Saldo akan bertambah kembali. Catatan tetap tersimpan untuk audit. Ini pembatalan pencatatan, bukan proses mengembalikan uang ke bank.</p><label><span>Alasan pembatalan</span><textarea v-model="voidReason" required minlength="5" maxlength="1000" rows="2"></textarea></label><div class="actions"><button type="button" :disabled="busy" @click="voidID = ''">Tutup</button><button type="submit" :disabled="busy">Simpan Pembatalan</button></div></form>
      <template v-if="intents.length"><h4>Tautan & Status Midtrans</h4><div class="gateway-list"><article v-for="item in intents" :key="item.id"><div><strong>{{ money(item.amount) }} · {{ intentState(item) }}</strong><small>{{ item.environment }} · {{ item.gateway_order_id }}</small><small>Status provider: {{ item.gateway_status || 'Menunggu dipilihnya metode bayar' }}</small></div><div class="actions"><a v-if="safeLink(item) && activeIntent(item)" :href="safeLink(item)" target="_blank" rel="noopener noreferrer">Buka Midtrans</a><button v-if="safeLink(item) && activeIntent(item)" @click="copyLink(item)">Salin Tautan</button><button :disabled="busy" @click="act(syncFinancePayment, item.id)">Cek Status</button><button v-if="item.state === 'CREATING'" :disabled="busy" @click="act(recoverFinancePayment, item.id)">Pulihkan Tautan</button><button v-if="activeIntent(item)" :disabled="busy" @click="act(cancelFinancePayment, item.id)">Batalkan Sesi</button></div></article></div></template>
      <button v-if="events.length" class="audit-toggle" @click="showAudit = !showAudit">{{ showAudit ? 'Sembunyikan' : 'Lihat' }} jejak pencatatan ({{ events.length }})</button>
      <ul v-if="showAudit" class="audit"><li v-for="event in events" :key="event.id"><strong>{{ event.action }}</strong> · {{ date(event.created_at) }}<small>{{ event.actor }} · {{ event.detail }}</small></li></ul>
    </template>
  </section>
</template>

<style scoped>
.payment-panel{color:var(--fas-text);min-width:0}.section-heading,.actions{display:flex;gap:.55rem;flex-wrap:wrap;align-items:center}.section-heading{justify-content:space-between;margin-bottom:1rem}h3{font-size:1rem;font-weight:800}h4{font-size:.88rem;font-weight:800;margin:1.2rem 0 .7rem}.section-heading p,.muted{font-size:.76rem;color:var(--fas-muted);line-height:1.6;margin:.4rem 0}.actions button,.actions a,.audit-toggle{display:inline-flex;align-items:center;justify-content:center;min-height:2.85rem;padding:.55rem .8rem;border:1px solid var(--fas-border-strong);border-radius:.8rem;background:var(--fas-card);color:var(--fas-text-soft);font-size:.76rem;font-weight:700;text-decoration:none;cursor:pointer}.actions .primary{background:var(--fas-accent);border-color:var(--fas-accent);color:white}button:disabled{opacity:.5;cursor:not-allowed}button:focus-visible,a:focus-visible,input:focus-visible,textarea:focus-visible{outline:2px solid var(--fas-accent);outline-offset:2px}.balances{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));border:1px solid var(--fas-accent-border);background:var(--fas-accent-soft);border-radius:1rem;gap:.8rem;padding:1rem}.balances span{font-size:.73rem;color:var(--fas-muted)}.balances strong{display:block;font-size:1.05rem;margin-top:.4rem;overflow-wrap:anywhere}.notice{font-size:.78rem;line-height:1.6;border:1px solid var(--fas-border);border-radius:.85rem;padding:.8rem;margin:.8rem 0;background:var(--fas-elevated)}.danger{color:var(--fas-danger);background:var(--fas-danger-soft)}.success{color:var(--fas-accent-text);background:var(--fas-accent-soft)}.tools{margin:1rem 0}.payment-form{padding:1rem;background:var(--fas-elevated);border:1px solid var(--fas-border);border-radius:1rem;margin:1rem 0}.payment-form h4{margin-top:0}fieldset{border:0;min-width:0;padding:0;margin:0}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.85rem}.wide{grid-column:1/-1}label{display:flex;flex-direction:column;gap:.45rem;min-width:0;font-size:.76rem;color:var(--fas-text-soft)}input,textarea{width:100%;min-height:2.85rem;background:var(--fas-input);border:1px solid var(--fas-border-strong);border-radius:.8rem;padding:.65rem .8rem;color:var(--fas-text);font-size:.82rem}textarea{resize:vertical}.table-scroll{overflow:auto;max-width:100%}table{width:100%;min-width:760px;font-size:.76rem;border-collapse:collapse}th,td{border-bottom:1px solid var(--fas-border);padding:.75rem;text-align:left;vertical-align:top}th{color:var(--fas-muted);font-size:.7rem}small{display:block;color:var(--fas-muted);font-size:.68rem;line-height:1.5;margin-top:.3rem;overflow-wrap:anywhere}.money{text-align:right;white-space:nowrap}.badge{color:var(--fas-accent-text);background:var(--fas-accent-soft);padding:.25rem .5rem;border-radius:.5rem;font-size:.68rem;display:inline-block}.empty{text-align:center;padding:1.5rem;color:var(--fas-muted)}.gateway-list{display:grid;gap:.65rem}.gateway-list article{border:1px solid var(--fas-border);border-radius:.85rem;padding:.9rem;background:var(--fas-elevated)}.gateway-list strong{font-size:.8rem}.gateway-list .actions{margin-top:.7rem}.audit-toggle{margin-top:1rem}.audit{list-style:none;padding:0;margin-top:.8rem}.audit li{padding:.65rem 0;border-bottom:1px solid var(--fas-border);font-size:.72rem}@media(max-width:640px){.balances,.form-grid{grid-template-columns:1fr}.wide{grid-column:auto}.section-heading{align-items:flex-start}}
</style>
