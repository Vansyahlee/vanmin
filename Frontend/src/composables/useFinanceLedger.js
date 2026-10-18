import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { downloadFinanceXlsx, openFinancePrint, writeFinancePrint } from '@/utils/financeDocuments'
export const financeMoney = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 2 }).format(Number(value || 0))
export const financeDate = value => value ? new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeStyle: 'short', timeZone: 'Asia/Jakarta' }).format(new Date(value)) : '—'
export const financeToday = () => new Intl.DateTimeFormat('sv-SE', { timeZone: 'Asia/Jakarta', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date())
export const financeError = error => error.response?.data?.error || error.message || 'Data belum dapat diproses.'
export const useFinanceLedger = (fetcher, initial, document) => {
 const filters = reactive({ ...initial })
 const applied = ref({ ...initial })
 const data = ref([]), summary = ref({}), total = ref(0), metadata = ref({}), extra = ref({})
 const loading = ref(false), exporting = ref(false), ready = ref(false), error = ref(''), page = ref(0)
 const limit = 50
 let sequence = 0, disposed = false
 const load = async (reset = false, targetPage = page.value) => {
  if (exporting.value) return
  const seq = ++sequence, snapshot = { ...(reset ? filters : applied.value) }, wantedPage = reset ? 0 : targetPage
  loading.value = true; error.value = ''; ready.value = false
  try {
   const result = await fetcher({ ...snapshot, limit, offset: wantedPage * limit })
   if (disposed || seq !== sequence) return
   data.value = result.data || []; total.value = Number(result.meta?.total || 0)
   summary.value = result.summary || {}; metadata.value = result.meta || {}; extra.value = result
   applied.value = snapshot; page.value = wantedPage; ready.value = true
  } catch (failure) { if (!disposed && seq === sequence) error.value = financeError(failure) }
  finally { if (!disposed && seq === sequence) loading.value = false }
 }
 const description = snapshot => Object.entries(snapshot).map(([key, value]) => `${({ query: 'Cari', date_from: 'Dari', date_to: 'Sampai', status: 'Status', category: 'Kategori', method: 'Metode/sumber', account: 'Akun', direction: 'Arah', read: 'Dibaca', expense_kind: 'Jenis pengeluaran' })[key] || key}: ${value || 'Semua'}`).join(' · ')
 const report = async kind => {
  if (!ready.value || loading.value || exporting.value) return
  const snapshot = { ...applied.value }; let popup
  exporting.value = true; error.value = ''
  try {
   if (kind === 'print') popup = openFinancePrint()
   const rows = [], seen = new Set(); let expected = null, freshSummary = {}
   for (let offset = 0; ; offset += 200) {
    const result = await fetcher({ ...snapshot, limit: 200, offset })
    const count = Number(result.meta?.total || 0)
    if (expected === null) { expected = count; freshSummary = result.summary || {} }
    if (expected !== count) throw new Error('Data berubah saat ekspor. Perbarui tampilan lalu coba lagi.')
    for (const item of result.data || []) {
     if (seen.has(String(item.id))) throw new Error('Urutan data berubah saat ekspor. Silakan ulangi.')
     seen.add(String(item.id)); rows.push(item)
    }
    if (rows.length >= expected) break
    if (!result.data?.length || offset > 10000000) throw new Error('Rekap belum lengkap. Perbarui lalu coba lagi.')
   }
   const details = `${document.periodDescription ? document.periodDescription(snapshot) : description(snapshot)} · ${document.description?.(freshSummary) || ''} · Dicetak ${financeDate(new Date())}`
   if (kind === 'print') writeFinancePrint(popup, document.title, details, document.printHeadings || document.headings, document.printRows ? document.printRows(rows) : document.rows(rows))
   else downloadFinanceXlsx(document.filename, [[document.title], [details], document.headings, ...document.rows(rows)])
  } catch (failure) { popup?.close(); if (!disposed) error.value = financeError(failure) }
  finally { if (!disposed) exporting.value = false }
 }
 const canPrevious = computed(() => page.value > 0 && !loading.value && !exporting.value)
 const canNext = computed(() => (page.value + 1) * limit < total.value && !loading.value && !exporting.value)
 onMounted(() => load(true))
 onBeforeUnmount(() => { disposed = true; sequence++ })
 return { filters, applied, data, summary, total, metadata, extra, loading, exporting, ready, error, page, limit, load, report, description, canPrevious, canNext }
}
