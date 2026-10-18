<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { useModalDialog } from '@/composables/useModalDialog'
import {
  getFinanceExpenses,
  getFinanceCash,
  createFinanceExpense,
  voidFinanceExpense,
} from '@/services/finance'
import {
  downloadFinanceXlsx,
  openFinancePrint,
  writeFinancePrint,
} from '@/utils/financeDocuments'

import {
  monthDates,
  reportCents,
  reportDecimal,
  reportMoney,
} from '@/utils/financeReporting'
import { writeExpenseReceipt } from '@/utils/expenseReceipt'
import FasSelect from '@/components/ui/FasSelect.vue'
import {
  useFinanceLedger,
  financeMoney as money,
  financeDate as date,
  financeToday,
  financeError,
} from '@/composables/useFinanceLedger'
import '@/assets/finance-ledger.css'
import {
  salaryNetAmount,
  expenseKindLabel,
  expenseDetailsRows,
} from '@/utils/expenseForms'

const categories = [
  'BBM & Transportasi',
  'Listrik',
  'Internet & Telepon',
  'Sewa',
  'Perlengkapan',
  'Perawatan',
  'Pembelian Barang',
  'Pajak & Administrasi',
  'Lainnya',
]
const methodLabel = (value) => (value === 'CASH' ? 'Tunai' : 'Transfer bank')
const balanceLabel = (value) => {
  if (value == null) return 'Belum tersedia'
  try {
    return reportMoney(value)
  } catch {
    return 'Periksa nominal'
  }
}
function periodLabel(snapshot) {
  const from = snapshot.date_from,
    to = snapshot.date_to
  const today = financeToday(),
    year = today.slice(0, 4),
    month = today.slice(0, 7)
  let title = 'Data pengeluaran semua tanggal'
  if (from === year + '-01-01' && to === year + '-12-31')
    title = 'Data pengeluaran tahun ini (' + year + ')'
  else if (from === month + '-01' && to === monthDates(month).date_to)
    title = 'Data pengeluaran bulan ini (' + month + ')'
  else if (from && from === to) title = 'Data pengeluaran tanggal ' + from
  else if (from || to)
    title =
      'Data pengeluaran ' +
      (from || 'awal pencatatan') +
      ' sampai ' +
      (to || 'terbaru')
  const descriptions = [
    snapshot.query && 'Pencarian: ' + snapshot.query,
    snapshot.expense_kind &&
      'Jenis: ' + expenseKindLabel(snapshot.expense_kind),
    snapshot.category && 'Kategori: ' + snapshot.category,
    snapshot.method && 'Metode: ' + methodLabel(snapshot.method),
    'Status: ' +
      (snapshot.status === 'ALL'
        ? 'Semua'
        : snapshot.status === 'VOID'
          ? 'Dibatalkan'
          : 'Tercatat'),
  ].filter(Boolean)
  return title + ' · ' + descriptions.join(' · ')
}
const {
  filters,
  applied,
  data,
  summary,
  total,
  loading,
  exporting,
  ready,
  error,
  page,
  load,
  report,
  canPrevious,
  canNext,
} = useFinanceLedger(
  getFinanceExpenses,
  {
    query: '',
    ...monthDates(financeToday().slice(0, 7)),
    status: 'POSTED',
    category: '',
    method: '',
    expense_kind: '',
  },
  {
    title: 'PT FAS — Rekap Pengeluaran',
    filename: 'PTFAS_Pengeluaran.xlsx',
    headings: [
      'Nomor',
      'Tanggal',
      'Kategori',
      'Penerima',
      'Uraian',
      'Nominal',
      'Akun kas',
      'Saldo sebelum transaksi',
      'Saldo sesudah transaksi',
      'Metode',
      'Referensi',
      'Bukti HTTPS',
      'Status',
      'Dicatat oleh',
      'Waktu catat',
      'Alasan batal',
      'Dibatalkan oleh',
      'Waktu batal',
      'Jenis',
      'Periode Gaji',
      'Kode Karyawan',
      'Gaji Pokok',
      'Tunjangan',
      'Potongan',
      'Invoice Supplier',
      'Tanggal Invoice Supplier',
      'Referensi PO',
    ],
    rows: (items) =>
      items.map((p) => [
        p.expense_number,
        p.expense_date,
        p.category,
        p.payee,
        p.description,
        Number(p.amount),
        p.cash_account || '',
        p.balance_before ?? '',
        p.balance_after ?? '',
        p.method,
        p.reference_number,
        p.proof_url,
        p.status,
        p.created_by_name,
        date(p.created_at),
        p.void_reason,
        p.voided_by,
        date(p.voided_at),
        expenseKindLabel(p.expense_kind),
        p.details?.payroll_period,
        p.details?.employee_code,
        p.expense_kind === 'SALARY' ? Number(p.details?.basic_salary) : '',
        p.expense_kind === 'SALARY' ? Number(p.details?.allowance) : '',
        p.expense_kind === 'SALARY' ? Number(p.details?.deduction) : '',
        p.details?.supplier_invoice_number,
        p.details?.supplier_invoice_date,
        p.details?.purchase_reference,
      ]),
    printHeadings: [
      'Nomor / Tanggal',
      'Jenis',
      'Penerima',
      'Rincian',
      'Nominal',
      'Metode / Referensi',
      'Saldo sebelum → sesudah',
      'Status',
    ],
    printRows: (items) =>
      items.map((p) => [
        p.expense_number + ' / ' + p.expense_date,
        expenseKindLabel(p.expense_kind),
        p.payee,
        [
          p.description,
          ...expenseDetailsRows(p, money).map(
            ([label, value]) => label + ': ' + value,
          ),
        ].join(' · '),
        money(p.amount),
        methodLabel(p.method) + ' / ' + (p.reference_number || '—'),
        balanceLabel(p.balance_before) + ' → ' + balanceLabel(p.balance_after),
        p.status === 'POSTED' ? 'Tercatat' : 'Dibatalkan',
      ]),
    periodDescription: periodLabel,
    description: (s) =>
      `Pengeluaran tercatat ${money(s.posted_amount)} · Dibatalkan ${money(s.void_amount)} · Saldo buku bank/tunai awal ${balanceLabel(s.cash_opening)} · Masuk ${balanceLabel(s.cash_in)} · Keluar ${balanceLabel(s.cash_out)} · Akhir ${balanceLabel(s.cash_closing)} (saldo mengikuti metode/tanggal, bukan pencarian atau kategori)`,
  },
)
const dialog = ref(''),
  detail = ref(null),
  busy = ref(false),
  formError = ref(''),
  message = ref(''),
  submitted = ref(null),
  voidReason = ref('')
const form = reactive({
  expense_kind: 'GENERAL',
  employee_code: '',
  payroll_period: financeToday().slice(0, 7),
  basic_salary: '',
  allowance: '',
  deduction: '',
  supplier_invoice_number: '',
  supplier_invoice_date: '',
  purchase_reference: '',
  expense_date: financeToday(),
  category: categories[0],
  custom_category: '',
  payee: '',
  description: '',
  amount: '',
  method: 'CASH',
  reference_number: '',
  proof_url: '',
})
const cashPreview = ref(null),
  cashPreviewBusy = ref(false),
  cashPreviewError = ref('')
let cashSequence = 0
const refreshCashPreview = async () => {
  const sequence = ++cashSequence
  cashPreview.value = null
  cashPreviewError.value = ''
  cashPreviewBusy.value = true
  try {
    const result = await getFinanceCash({
      account: form.method === 'CASH' ? 'CASH' : 'BANK',
      date_to: financeToday(),
      limit: 1,
      offset: 0,
    })
    if (sequence === cashSequence)
      cashPreview.value = result.summary?.closing ?? null
  } catch (failure) {
    if (sequence === cashSequence)
      cashPreviewError.value = financeError(failure)
  } finally {
    if (sequence === cashSequence) cashPreviewBusy.value = false
  }
}
watch(
  () => [dialog.value, form.method],
  () => {
    if (dialog.value === 'create') refreshCashPreview()
    else {
      cashSequence++
      cashPreviewBusy.value = false
      cashPreview.value = null
    }
  },
)
const previewAfter = computed(() => {
  try {
    const amount =
      form.expense_kind === 'SALARY' ? salaryNet.value : form.amount
    return cashPreview.value == null || !amount || reportCents(amount) <= 0n
      ? null
      : reportDecimal(reportCents(cashPreview.value) - reportCents(amount))
  } catch {
    return null
  }
})
const usePeriod = (kind) => {
  const today = financeToday()
  if (kind === 'month') Object.assign(filters, monthDates(today.slice(0, 7)))
  if (kind === 'year')
    Object.assign(filters, {
      date_from: today.slice(0, 4) + '-01-01',
      date_to: today.slice(0, 4) + '-12-31',
    })
  if (kind === 'today')
    Object.assign(filters, { date_from: today, date_to: today })
  if (kind === 'all') Object.assign(filters, { date_from: '', date_to: '' })
  load(true)
}
const salaryNet = computed(() =>
  salaryNetAmount(form.basic_salary, form.allowance, form.deduction),
)
const detailTitle = computed(() =>
  detail.value?.expense_kind === 'SALARY'
    ? 'Slip Gaji — Bukti Pembayaran'
    : detail.value?.expense_kind === 'SUPPLIER'
      ? 'Bukti Pembayaran Supplier'
      : 'Bukti Pengeluaran',
)
const locked = computed(() => busy.value || Boolean(submitted.value))
const close = () => {
  if (!busy.value) dialog.value = ''
}
useModalDialog(() => (dialog.value ? '#expense-dialog' : ''), close)
const start = (kind = 'GENERAL') => {
  if (!['GENERAL', 'SALARY', 'SUPPLIER'].includes(kind)) kind = 'GENERAL'
  if (submitted.value) {
    dialog.value = 'create'
    return
  }
  Object.assign(form, {
    expense_kind: kind,
    employee_code: '',
    payroll_period: financeToday().slice(0, 7),
    basic_salary: '',
    allowance: '',
    deduction: '',
    supplier_invoice_number: '',
    supplier_invoice_date: '',
    purchase_reference: '',
    expense_date: financeToday(),
    category: categories[0],
    custom_category: '',
    payee: '',
    description: '',
    amount: '',
    method: 'CASH',
    reference_number: '',
    proof_url: '',
  })
  if (kind === 'SALARY') {
    form.category = 'Gaji & Honor'
    form.description = 'Pembayaran gaji karyawan'
  }
  if (kind === 'SUPPLIER') {
    form.category = 'Pembayaran Supplier'
    form.description = 'Pembayaran invoice supplier'
  }
  formError.value = ''
  dialog.value = 'create'
}
const save = async () => {
  if (busy.value) return
  formError.value = ''
  if (!submitted.value) {
    const category =
      form.expense_kind === 'SALARY'
        ? 'Gaji & Honor'
        : form.expense_kind === 'SUPPLIER'
          ? 'Pembayaran Supplier'
          : (form.category === 'Lainnya'
              ? form.custom_category
              : form.category
            ).trim()
    const amount =
      form.expense_kind === 'SALARY' ? salaryNet.value : form.amount
    const details =
      form.expense_kind === 'SALARY'
        ? {
            employee_code: form.employee_code.trim(),
            payroll_period: form.payroll_period,
            basic_salary: form.basic_salary,
            allowance: form.allowance || '0',
            deduction: form.deduction || '0',
          }
        : form.expense_kind === 'SUPPLIER'
          ? {
              supplier_invoice_number: form.supplier_invoice_number.trim(),
              supplier_invoice_date: form.supplier_invoice_date,
              purchase_reference: form.purchase_reference.trim(),
            }
          : {}
    if (
      form.expense_kind === 'SALARY' &&
      (!salaryNet.value || !/^\d{4}-\d{2}$/.test(form.payroll_period))
    ) {
      formError.value =
        'Isi periode dan gaji pokok. Gaji bersih harus positif setelah tunjangan dan potongan.'
      return
    }
    if (form.expense_kind === 'SUPPLIER' && !details.supplier_invoice_number) {
      formError.value =
        'Nomor invoice supplier wajib diisi agar pembayaran dapat ditelusuri.'
      return
    }
    if (
      !category ||
      !form.payee.trim() ||
      form.description.trim().length < 3 ||
      !/^\d+(\.\d{1,2})?$/.test(amount) ||
      Number(amount) <= 0 ||
      !form.expense_date ||
      form.expense_date > financeToday() ||
      (form.method === 'BANK_TRANSFER' && !form.reference_number.trim())
    ) {
      formError.value =
        'Lengkapi isian wajib. Nominal tanpa titik ribuan, contoh 150000 atau 150000.50.'
      return
    }
    if (form.proof_url.trim()) {
      try {
        const u = new URL(form.proof_url.trim())
        if (u.protocol !== 'https:' || u.username || u.password)
          throw new Error()
      } catch {
        formError.value = 'Tautan bukti harus berupa URL HTTPS yang valid.'
        return
      }
    }
    submitted.value = {
      request_key: crypto.randomUUID(),
      expense_date: form.expense_date,
      category,
      payee: form.payee.trim(),
      description: form.description.trim(),
      amount,
      expense_kind: form.expense_kind,
      details,
      method: form.method,
      reference_number: form.reference_number.trim(),
      proof_url: form.proof_url.trim(),
    }
  }
  busy.value = true
  try {
    await createFinanceExpense({ ...submitted.value })
    submitted.value = null
    dialog.value = ''
    message.value = 'Pengeluaran tersimpan dan mutasi kas tercatat.'
    await load()
  } catch (failure) {
    formError.value = financeError(failure)
    if (
      failure.response?.status >= 400 &&
      failure.response.status < 500 &&
      failure.response.status !== 409
    )
      submitted.value = null
  } finally {
    busy.value = false
  }
}
const inspect = (item) => {
  detail.value = item
  voidReason.value = ''
  formError.value = ''
  dialog.value = 'detail'
}
const voidExpense = async () => {
  if (busy.value || exporting.value || voidReason.value.trim().length < 5)
    return
  busy.value = true
  formError.value = ''
  try {
    await voidFinanceExpense(detail.value.id, voidReason.value.trim())
    dialog.value = ''
    message.value =
      'Pembatalan tersimpan. Riwayat tetap ada dan mutasi pembalik kas tercatat.'
    await load()
  } catch (failure) {
    formError.value = financeError(failure)
  } finally {
    busy.value = false
  }
}
const detailRows = computed(() =>
  detail.value
    ? [
        ['Jenis', expenseKindLabel(detail.value.expense_kind)],
        ...expenseDetailsRows(detail.value, money),
        ['Nomor pengeluaran', detail.value.expense_number],
        ['Tanggal', detail.value.expense_date],
        ['Kategori', detail.value.category],
        ['Penerima', detail.value.payee],
        ['Uraian', detail.value.description],
        ['Nominal', money(detail.value.amount)],
        [
          'Akun kas',
          detail.value.cash_account === 'BANK'
            ? 'Bank'
            : detail.value.cash_account === 'CASH'
              ? 'Tunai'
              : 'Belum tersedia',
        ],
        ['Saldo sebelum transaksi', balanceLabel(detail.value.balance_before)],
        ['Saldo sesudah transaksi', balanceLabel(detail.value.balance_after)],
        ['Metode', detail.value.method],
        ['Referensi', detail.value.reference_number || '—'],
        [
          'Status',
          detail.value.status === 'POSTED' ? 'Tercatat' : 'Dibatalkan',
        ],
        ['Dicatat oleh', detail.value.created_by_name],
        ['Waktu catat', date(detail.value.created_at)],
        ['Bukti', detail.value.proof_url || 'Belum ditautkan'],
        ['Alasan batal', detail.value.void_reason || '—'],
        ['Dibatalkan oleh', detail.value.voided_by || '—'],
        ['Waktu batal', date(detail.value.voided_at)],
      ]
    : [],
)
const printDetail = () => {
  try {
    writeExpenseReceipt(openFinancePrint(), detail.value, detailRows.value)
  } catch (failure) {
    formError.value = financeError(failure)
  }
}
const excelDetail = () =>
  downloadFinanceXlsx('PTFAS_Bukti_Pengeluaran.xlsx', [
    [`PT FAS — ${detailTitle.value}`],
    ['Keterangan', 'Isi'],
    ...detailRows.value,
  ])
</script>
<template>
  <main class="finance-ledger expenses-page">
    <header class="expenses-heading">
      <div>
        <p class="eyebrow">Finance / Kas keluar</p>
        <h1>Pengeluaran</h1>
        <p class="expenses-subtitle">Pembayaran operasional, gaji karyawan, dan supplier.</p>
      </div>
      <button type="button" class="primary" :disabled="exporting" @click="start('GENERAL')">+ Catat pengeluaran</button>
    </header>

    <section class="expenses-types" aria-label="Kategori pencatatan">
      <button type="button" :disabled="exporting" @click="start('GENERAL')">
        <span class="type-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M5 3h14v18l-3-2-4 2-4-2-3 2V3zM9 8h6M9 12h6"/></svg></span>
        <span><strong>Operasional</strong><small>Listrik, BBM, sewa, dan lainnya</small></span>
        <span class="type-arrow">↗</span>
      </button>
      <button type="button" :disabled="exporting" @click="start('SALARY')">
        <span class="type-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><circle cx="12" cy="7" r="4"/><path d="M4 21v-2a8 8 0 0 1 16 0v2"/></svg></span>
        <span><strong>Gaji karyawan</strong><small>Catat gaji yang telah dibayarkan</small></span>
        <span class="type-arrow">↗</span>
      </button>
      <button type="button" :disabled="exporting" @click="start('SUPPLIER')">
        <span class="type-icon"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="m3 7 9-4 9 4-9 4-9-4ZM3 7v10l9 4 9-4V7M12 11v10"/></svg></span>
        <span><strong>Supplier</strong><small>Catat pembayaran tagihan barang</small></span>
        <span class="type-arrow">↗</span>
      </button>
    </section>

    <p v-if="error" class="message error" role="alert">{{ error }}</p>
    <p v-if="message" class="message" role="status">{{ message }}</p>

    <section class="expenses-kpis" aria-label="Ringkasan pengeluaran">
      <article class="expenses-kpi"><span>Pengeluaran tercatat</span><strong>{{ ready ? money(summary.posted_amount) : '—' }}</strong><small>Sesuai filter aktif</small></article>
      <article class="expenses-kpi"><span>Catatan pengeluaran</span><strong>{{ ready ? total : '—' }}</strong><small>Seluruh halaman hasil pencarian</small></article>
      <article class="expenses-kpi"><span>Saldo akhir buku kas</span><strong>{{ ready ? balanceLabel(summary.cash_closing) : '—' }}</strong><small>Catatan bank dan tunai</small></article>
      <article class="expenses-kpi"><span>Catatan dibatalkan</span><strong>{{ ready ? money(summary.void_amount) : '—' }}</strong><small>Tetap tersimpan untuk audit</small></article>
    </section>

    <section class="expenses-panel expenses-report-panel">
      <div class="expenses-panel-head">
        <div><h2>Riwayat Pengeluaran</h2><p>{{ periodLabel(applied) }}</p></div>
        <div class="expenses-head-actions">
          <button type="button" :disabled="loading || exporting" @click="load()">Perbarui</button>
          <button type="button" :disabled="!ready || loading || exporting" @click="report('excel')">Excel</button>
          <button type="button" :disabled="!ready || loading || exporting" @click="report('print')">Print / PDF</button>
        </div>
      </div>

      <div class="expenses-filters">
        <div class="expenses-period-shortcuts">
          <button type="button" :disabled="loading || exporting" @click="usePeriod('today')">Hari ini</button>
          <button type="button" :disabled="loading || exporting" @click="usePeriod('month')">Bulan ini</button>
          <button type="button" :disabled="loading || exporting" @click="usePeriod('year')">Tahun ini</button>
          <button type="button" :disabled="loading || exporting" @click="usePeriod('all')">Semua</button>
        </div>
        <form @submit.prevent="load(true)">
          <fieldset :disabled="loading || exporting">
            <div class="expenses-filter-grid">
              <label class="expenses-filter-search"><span>Cari transaksi</span><input v-model="filters.query" type="search" maxlength="200" placeholder="Nama, nomor, atau uraian" /></label>
              <label><span>Dari tanggal</span><input v-model="filters.date_from" type="date" /></label>
              <label><span>Sampai tanggal</span><input v-model="filters.date_to" type="date" /></label>
              <label><span>Jenis</span><FasSelect v-model="filters.expense_kind" :disabled="loading || exporting" aria-label="Jenis pengeluaran"><option value="">Semua jenis</option><option value="GENERAL">Operasional</option><option value="SALARY">Gaji</option><option value="SUPPLIER">Supplier</option></FasSelect></label>
              <label><span>Status</span><FasSelect v-model="filters.status" :disabled="loading || exporting" aria-label="Status pengeluaran"><option value="ALL">Semua</option><option value="POSTED">Tercatat</option><option value="VOID">Dibatalkan</option></FasSelect></label>
              <button class="primary" type="submit">Terapkan</button>
            </div>
            <details class="expenses-extra-filters">
              <summary>Filter lainnya</summary>
              <div class="expenses-extra-grid">
                <label><span>Kategori</span><input v-model="filters.category" maxlength="100" placeholder="Semua kategori" /></label>
                <label><span>Metode</span><FasSelect v-model="filters.method" :disabled="loading || exporting" aria-label="Metode"><option value="">Semua metode</option><option value="CASH">Tunai</option><option value="BANK_TRANSFER">Transfer bank</option></FasSelect></label>
              </div>
            </details>
          </fieldset>
        </form>
      </div>

      <div class="table-scroll">
        <table class="expenses-table">
          <thead><tr><th>Nomor / tanggal</th><th>Jenis</th><th>Penerima</th><th class="money">Nominal</th><th>Pembayaran</th><th>Status</th><th>Aksi</th></tr></thead>
          <tbody>
            <tr v-for="item in data" :key="item.id">
              <td><strong>{{ item.expense_number }}</strong><small>{{ item.expense_date }}</small></td>
              <td>{{ expenseKindLabel(item.expense_kind) }}<small>{{ item.category }}</small></td>
              <td>{{ item.payee }}<small>{{ item.description }}</small></td>
              <td class="money"><strong>{{ money(item.amount) }}</strong></td>
              <td>{{ methodLabel(item.method) }}<small>{{ item.reference_number || '—' }}</small></td>
              <td><span class="expenses-status" :class="{ 'is-void': item.status === 'VOID' }">{{ item.status === 'POSTED' ? 'Tercatat' : 'Dibatalkan' }}</span></td>
              <td><button type="button" :disabled="exporting" @click="inspect(item)">Lihat detail</button></td>
            </tr>
            <tr v-if="!data.length"><td colspan="7" class="empty">{{ loading ? 'Memuat pengeluaran…' : 'Belum ada pengeluaran sesuai filter.' }}</td></tr>
          </tbody>
        </table>
      </div>
      <footer class="expenses-pagination">
        <span>{{ total }} data · Halaman {{ page + 1 }}</span>
        <div class="actions"><button type="button" :disabled="!canPrevious" @click="load(false, page - 1)">Sebelumnya</button><button type="button" :disabled="!canNext" @click="load(false, page + 1)">Berikutnya</button></div>
      </footer>
    </section>

    <Teleport to="body">
      <div v-if="dialog" class="finance-ledger dialog-backdrop expenses-backdrop" @click.self="close">
        <section id="expense-dialog" class="dialog expenses-dialog" role="dialog" aria-modal="true" aria-labelledby="expense-title">
          <header class="expenses-dialog-head">
            <div><p class="eyebrow">Finance / Pembayaran</p><h2 id="expense-title">{{ dialog === 'create' ? `Catat ${expenseKindLabel(form.expense_kind)}` : detailTitle }}</h2></div>
            <button type="button" :disabled="busy" @click="close" aria-label="Tutup">Tutup</button>
          </header>

          <form v-if="dialog === 'create'" @submit.prevent="save">
            <div class="body expenses-dialog-body">
              <p v-if="submitted" class="message">Pencatatan belum terkonfirmasi. Gunakan Coba Lagi agar transaksi tidak tersimpan ganda.</p>
              <p v-if="formError" class="message error" role="alert">{{ formError }}</p>
              <fieldset :disabled="locked">
                <div class="expenses-form-grid">
                  <label class="expenses-span-two"><span>{{ form.expense_kind === 'SALARY' ? 'Nama karyawan' : form.expense_kind === 'SUPPLIER' ? 'Nama supplier' : 'Penerima pembayaran' }} *</span><input v-model="form.payee" required maxlength="200" autocomplete="off" placeholder="Nama penerima" /></label>
                  <label><span>Tanggal pembayaran *</span><input v-model="form.expense_date" type="date" min="2000-01-01" :max="financeToday()" required /></label>
                  <label v-if="form.expense_kind === 'SALARY'"><span>Periode gaji *</span><input v-model="form.payroll_period" type="month" required /></label>
                  <label v-else-if="form.expense_kind === 'GENERAL'"><span>Kategori *</span><FasSelect v-model="form.category" :disabled="locked" aria-label="Kategori pengeluaran"><option v-for="c in categories" :key="c" :value="c">{{ c }}</option></FasSelect></label>
                  <label v-if="form.expense_kind === 'GENERAL' && form.category === 'Lainnya'" class="expenses-span-two"><span>Nama kategori *</span><input v-model="form.custom_category" required maxlength="100" placeholder="Kategori lainnya" /></label>
                  <template v-if="form.expense_kind === 'SALARY'">
                    <label><span>Gaji pokok (Rp) *</span><input v-model="form.basic_salary" required inputmode="decimal" placeholder="Contoh: 4000000" /></label>
                    <div class="expenses-net-box"><span>Total dibayarkan</span><strong>{{ salaryNet ? money(salaryNet) : '—' }}</strong></div>
                  </template>
                  <label v-else><span>Nominal dibayar (Rp) *</span><input v-model="form.amount" required inputmode="decimal" pattern="[0-9]+(\.[0-9]{1,2})?" placeholder="Contoh: 150000" /></label>
                  <label v-if="form.expense_kind === 'SUPPLIER'"><span>No. invoice supplier *</span><input v-model="form.supplier_invoice_number" required maxlength="150" placeholder="Nomor invoice" /></label>
                  <label><span>Metode *</span><FasSelect v-model="form.method" :disabled="locked" aria-label="Metode pembayaran"><option value="CASH">Tunai</option><option value="BANK_TRANSFER">Transfer bank</option></FasSelect></label>
                  <label v-if="form.method === 'BANK_TRANSFER'"><span>Referensi transfer *</span><input v-model="form.reference_number" required maxlength="150" placeholder="Nomor transfer bank" /></label>
                  <label class="expenses-span-two"><span>Keterangan *</span><textarea v-model="form.description" required minlength="3" maxlength="2000" rows="2" placeholder="Keterangan pembayaran" /></label>
                </div>
                <details class="expenses-form-more">
                  <summary>Rincian tambahan</summary>
                  <div class="expenses-form-grid">
                    <template v-if="form.expense_kind === 'SALARY'">
                      <label><span>Tunjangan (Rp)</span><input v-model="form.allowance" inputmode="decimal" placeholder="0" /></label>
                      <label><span>Potongan (Rp)</span><input v-model="form.deduction" inputmode="decimal" placeholder="0" /></label>
                      <label class="expenses-span-two"><span>Kode karyawan (opsional)</span><input v-model="form.employee_code" maxlength="80" placeholder="Kode internal" /></label>
                    </template>
                    <template v-if="form.expense_kind === 'SUPPLIER'">
                      <label><span>Tanggal invoice</span><input v-model="form.supplier_invoice_date" type="date" /></label>
                      <label><span>Referensi PO</span><input v-model="form.purchase_reference" maxlength="150" /></label>
                    </template>
                    <label v-if="form.method === 'CASH'"><span>Nomor kuitansi</span><input v-model="form.reference_number" maxlength="150" /></label>
                    <label class="expenses-span-two"><span>Tautan bukti (HTTPS)</span><input v-model="form.proof_url" type="url" maxlength="2000" placeholder="https://" /></label>
                  </div>
                </details>
              </fieldset>
              <div class="expenses-balance-preview">
                <div><span>Saldo buku {{ form.method === 'CASH' ? 'tunai' : 'bank' }}</span><strong>{{ cashPreviewBusy ? 'Memuat…' : balanceLabel(cashPreview) }}</strong></div>
                <div><span>Nominal dicatat</span><strong>{{ balanceLabel(form.expense_kind === 'SALARY' ? salaryNet || null : form.amount || null) }}</strong></div>
                <div><span>Perkiraan saldo setelah catat</span><strong>{{ balanceLabel(previewAfter) }}</strong></div>
                <button type="button" :disabled="cashPreviewBusy || busy" @click="refreshCashPreview">Perbarui saldo</button>
                <p v-if="cashPreviewError" class="message error">{{ cashPreviewError }}</p>
              </div>
              <p class="expenses-disclaimer">Pencatatan ini tidak mengirim atau memindahkan uang. Saldo buku bukan saldo bank terverifikasi.</p>
            </div>
            <footer class="expenses-dialog-footer"><button type="button" :disabled="busy" @click="close">Batal</button><button class="primary" type="submit" :disabled="busy">{{ busy ? 'Menyimpan…' : submitted ? 'Coba Lagi' : 'Simpan pengeluaran' }}</button></footer>
          </form>

          <div v-else class="body expenses-detail-body">
            <article class="expenses-receipt-top">
              <p>{{ detail.expense_number }}</p>
              <strong>{{ money(detail.amount) }}</strong>
              <span>{{ detail.expense_date }} · {{ detail.payee }}</span>
              <span class="expenses-status" :class="{ 'is-void': detail.status === 'VOID' }">{{ detail.status === 'POSTED' ? 'Tercatat' : 'Dibatalkan' }}</span>
            </article>
            <dl class="detail-grid"><div v-for="[label, value] in detailRows" :key="label"><dt>{{ label }}</dt><dd>{{ value }}</dd></div></dl>
            <div class="actions expenses-receipt-actions">
              <button type="button" @click="printDetail">{{ detail.expense_kind === 'SALARY' ? 'Print / PDF slip gaji' : 'Print / PDF bukti' }}</button>
              <button type="button" @click="excelDetail">Excel</button>
              <a v-if="detail.proof_url" class="button" :href="detail.proof_url" target="_blank" rel="noopener noreferrer">Buka bukti</a>
            </div>
            <p v-if="formError" class="message error" role="alert">{{ formError }}</p>
            <details v-if="detail.status === 'POSTED'" class="expenses-form-more expenses-void-details">
              <summary>Batalkan pencatatan</summary>
              <form @submit.prevent="voidExpense">
                <p>Catatan dan jejak audit tetap tersimpan. Pembatalan tidak melakukan transfer atau refund.</p>
                <label><span>Alasan pembatalan *</span><textarea v-model="voidReason" required minlength="5" maxlength="1000" :disabled="busy" rows="3" /></label>
                <button class="danger" type="submit" :disabled="busy || exporting || voidReason.trim().length < 5">{{ busy ? 'Memproses…' : 'Konfirmasi pembatalan' }}</button>
              </form>
            </details>
          </div>
        </section>
      </div>
    </Teleport>
  </main>
</template>
<style scoped>
.expenses-page {padding:1.65rem clamp(1rem,2.3vw,2.1rem) 3rem;color:var(--fas-text)}
.expenses-heading{display:flex;align-items:center;justify-content:space-between;gap:1rem;flex-wrap:wrap;margin-bottom:1.35rem}
.expenses-heading h1{font-size:clamp(1.55rem,2.4vw,2rem);margin:.25rem 0;font-weight:800;letter-spacing:-.04em}
.expenses-subtitle{margin:.3rem 0 0;color:var(--fas-muted);font-size:.82rem}
.expenses-types{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.85rem;margin-bottom:1.15rem}
.finance-ledger .expenses-types button{padding:1rem 1.1rem;border:1px solid var(--fas-border);border-radius:1.15rem;background:var(--fas-card);display:flex;text-align:left;align-items:center;gap:.85rem;min-width:0;min-height:5.5rem;box-shadow:var(--fas-shadow-soft);font-weight:400;transition:transform .2s,border-color .2s}
.expenses-types button:hover{transform:translateY(-2px);border-color:var(--fas-accent-border)}
.type-icon{flex-shrink:0;display:grid;place-items:center;width:2.65rem;height:2.65rem;border-radius:.9rem;background:var(--fas-accent-soft);color:var(--fas-accent-text)}
.type-icon svg{width:1.35rem;height:1.35rem}
.expenses-types button>span:nth-child(2){display:grid;min-width:0;gap:.35rem}
.expenses-types strong{font-size:.85rem;color:var(--fas-text)}
.expenses-types small{font-size:.71rem;color:var(--fas-muted);line-height:1.5}
.type-arrow{font-size:1.05rem;color:var(--fas-accent-text);margin-left:auto}
.expenses-kpis{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:.85rem;margin:1rem 0 1.1rem}
.expenses-kpi{background:var(--fas-card);border:1px solid var(--fas-border);border-radius:1.15rem;padding:1.1rem 1.2rem;display:flex;flex-direction:column;gap:.9rem;min-width:0;min-height:8.6rem;box-shadow:var(--fas-shadow-soft)}
.expenses-kpi span{font-size:.73rem;color:var(--fas-muted);font-weight:700}
.expenses-kpi strong{font-size:clamp(1.14rem,1.9vw,1.6rem);line-height:1.25;font-weight:800;letter-spacing:-.035em;overflow-wrap:anywhere;color:var(--fas-text)}
.expenses-kpi small{margin-top:auto;font-size:.7rem}
.expenses-panel{background:var(--fas-card);border:1px solid var(--fas-border);border-radius:1.2rem;box-shadow:var(--fas-shadow-soft);overflow:hidden;min-width:0}
.expenses-panel-head{display:flex;justify-content:space-between;align-items:center;gap:1rem;flex-wrap:wrap;padding:1.2rem;border-bottom:1px solid var(--fas-border)}
.expenses-panel-head h2{font-size:1.05rem;font-weight:770}
.expenses-panel-head p{font-size:.74rem;color:var(--fas-muted);margin:.35rem 0 0;line-height:1.5}
.expenses-head-actions{display:flex;gap:.55rem;flex-wrap:wrap}
.expenses-filters{padding:1rem 1.2rem;border-bottom:1px solid var(--fas-border)}
.expenses-period-shortcuts{display:flex;gap:.4rem;flex-wrap:wrap;margin-bottom:.85rem}
.finance-ledger .expenses-period-shortcuts button{min-height:2.3rem;border-radius:.65rem;font-size:.74rem;padding:.4rem .65rem}
.expenses-filters fieldset{border:0;min-width:0;margin:0;padding:0}
.expenses-filter-grid{display:grid;grid-template-columns:minmax(10rem,2fr) repeat(2,minmax(9rem,1fr)) minmax(9rem,1fr) minmax(9rem,1fr) auto;gap:.65rem;align-items:end}
.expenses-filter-grid label,.expenses-extra-grid label{display:grid;gap:.35rem;min-width:0;font-size:.72rem;color:var(--fas-text-soft);font-weight:700}
.expenses-filter-grid input,.expenses-extra-grid input{min-width:0;width:100%}
.expenses-filter-grid button{white-space:nowrap}
.expenses-extra-filters{margin-top:.85rem}
.expenses-extra-filters summary{font-size:.73rem;color:var(--fas-accent-text);cursor:pointer;font-weight:700}
.expenses-extra-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.8rem;max-width:34rem;margin-top:.75rem}
.expenses-table{width:100%;min-width:57rem}
.expenses-table td{vertical-align:middle}
.expenses-table td strong{font-size:.77rem}
.expenses-table td small{display:block;max-width:18rem;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;margin-top:.32rem}
.expenses-table td.money strong{font-size:.8rem}
.expenses-status{display:inline-flex;border-radius:999px;background:var(--fas-success-soft,var(--fas-accent-soft));color:var(--fas-success,var(--fas-accent-text));padding:.35rem .65rem;font-size:.7rem;font-weight:760;white-space:nowrap}
.expenses-status.is-void{background:var(--fas-danger-soft,var(--fas-accent-soft));color:var(--fas-danger,#c34148)}
.expenses-pagination{display:flex;justify-content:space-between;align-items:center;gap:1rem;padding:.95rem 1.2rem;border-top:1px solid var(--fas-border);flex-wrap:wrap;font-size:.74rem;color:var(--fas-muted)}
.expenses-backdrop{z-index:2000}
.finance-ledger .expenses-dialog{width:min(100%,44rem);max-height:min(91vh,59rem);border-radius:1.3rem;overflow:hidden;display:flex;flex-direction:column}
.expenses-dialog-head{display:flex;align-items:center;justify-content:space-between;gap:1rem;padding:1.2rem 1.4rem;border-bottom:1px solid var(--fas-border)}
.expenses-dialog-head h2{font-size:1.23rem;margin:.3rem 0 0;font-weight:780}
.expenses-dialog>form{display:flex;flex-direction:column;min-height:0}
.expenses-dialog-body{overflow-y:auto;min-height:0;padding:1.3rem!important}
.expenses-dialog-body fieldset{border:0;padding:0;margin:0;min-width:0}
.expenses-form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:.95rem}
.expenses-form-grid label,.expenses-void-details label{display:grid;gap:.4rem;min-width:0;font-size:.78rem;font-weight:700;color:var(--fas-text-soft)}
.expenses-form-grid label>span,.expenses-void-details label>span{display:block}
.expenses-form-grid input,.expenses-form-grid textarea,.expenses-void-details textarea{width:100%;min-width:0}
.expenses-span-two{grid-column:1/-1}
.expenses-net-box{display:flex;flex-direction:column;justify-content:center;gap:.5rem;border:1px solid var(--fas-accent-border);border-radius:.9rem;background:var(--fas-accent-soft);padding:.85rem 1rem;min-width:0}
.expenses-net-box span{font-size:.75rem;color:var(--fas-text-soft)}
.expenses-net-box strong{font-size:1.1rem;font-weight:800;overflow-wrap:anywhere}
.expenses-form-more{margin-top:1.2rem;border:1px solid var(--fas-border);border-radius:.9rem;background:var(--fas-elevated);padding:.85rem 1rem}
.expenses-form-more summary{cursor:pointer;font-size:.8rem;font-weight:750;color:var(--fas-text-soft)}
.expenses-form-more[open] summary{margin-bottom:1rem}
.expenses-balance-preview{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.75rem;padding:1rem;border:1px solid var(--fas-border);border-radius:.95rem;background:var(--fas-elevated);margin-top:1rem;align-items:center}
.expenses-balance-preview>div{display:grid;gap:.42rem;min-width:0}
.expenses-balance-preview>div span{color:var(--fas-muted);font-size:.68rem}
.expenses-balance-preview>div strong{font-size:.85rem;overflow-wrap:anywhere}
.expenses-balance-preview>button{grid-column:1/-1;justify-self:start;min-height:2.1rem;padding:.35rem .7rem;font-size:.7rem}
.expenses-balance-preview>p{grid-column:1/-1}
.expenses-disclaimer{font-size:.73rem;color:var(--fas-muted);line-height:1.6;margin:.85rem 0 0}
.expenses-dialog-footer{display:flex;justify-content:flex-end;gap:.6rem;padding:1rem 1.3rem;border-top:1px solid var(--fas-border)}
.expenses-detail-body{overflow:auto;min-height:0;padding:1.25rem!important}
.expenses-receipt-top{display:flex;align-items:center;flex-direction:column;gap:.55rem;padding:1.4rem;border:1px solid var(--fas-accent-border);border-radius:1rem;background:var(--fas-accent-soft);margin-bottom:1rem;text-align:center}
.expenses-receipt-top p{margin:0;color:var(--fas-muted);font-size:.78rem}
.expenses-receipt-top strong{font-size:1.7rem;font-weight:800}
.expenses-receipt-top>span:not(.expenses-status){font-size:.76rem;color:var(--fas-muted)}
.expenses-receipt-actions{margin:1rem 0;flex-wrap:wrap}
.expenses-void-details{background:var(--fas-card)}
.expenses-void-details form{display:grid;gap:.85rem}
.expenses-void-details p{font-size:.75rem;color:var(--fas-muted);margin:0}
.expenses-void-details button{justify-self:start}
@media (max-width:1350px){.expenses-filter-grid{grid-template-columns:repeat(3,minmax(0,1fr))}.expenses-filter-search{grid-column:span 2}}
@media (max-width:950px){.expenses-kpis{grid-template-columns:repeat(2,minmax(0,1fr))}.expenses-types{grid-template-columns:1fr}}
@media (max-width:580px){.expenses-page{padding:1rem .8rem 2rem}.expenses-kpis{gap:.55rem}.expenses-kpi{min-height:8rem;padding:.9rem}.expenses-kpi strong{font-size:1.1rem}.expenses-filter-grid{grid-template-columns:1fr 1fr}.expenses-filter-search{grid-column:1/-1}.expenses-filter-grid button{grid-column:1/-1}.expenses-filters{padding:.9rem}.expenses-form-grid{grid-template-columns:1fr}.expenses-span-two{grid-column:auto}.expenses-balance-preview{grid-template-columns:1fr}.expenses-dialog-head{padding:1rem}.expenses-dialog-body{padding:1rem!important}}
</style>
