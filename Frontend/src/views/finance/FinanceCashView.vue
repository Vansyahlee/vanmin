<script setup>
import { useRouter } from 'vue-router'
import { getFinanceCash } from '@/services/finance'

import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import FasSelect from '@/components/ui/FasSelect.vue'
import { useFinanceLedger, financeMoney as money, financeDate as date, financeToday, financeError } from '@/composables/useFinanceLedger'
import '@/assets/finance-ledger.css'

const router = useRouter()
const accountLabel = value => ({ CASH: 'Tunai', BANK: 'Bank', MIDTRANS: 'Midtrans belum direkonsiliasi', LEGACY: 'Pembayaran lama / belum diklasifikasi' })[value] || value
const movementDate = item => item.kind === 'EXPENSE' ? new Intl.DateTimeFormat('id-ID', { dateStyle: 'medium', timeZone: 'Asia/Jakarta' }).format(new Date(item.occurred_at)) : date(item.occurred_at)
const kindLabel = value => ({ PAYMENT_CONFIRMED: 'Pembayaran masuk', PAYMENT_IMPORTED: 'Impor pembayaran lama', PAYMENT_REFUND: 'Koreksi refund', PAYMENT_VOID: 'Pembatalan pembayaran', EXPENSE: 'Pengeluaran', EXPENSE_VOID: 'Pembalik pengeluaran' })[value] || value
const { filters, applied, data, summary, total, loading, exporting, ready, error, page, load, report, description, canPrevious, canNext } = useFinanceLedger(getFinanceCash,
 { query: '', date_from: '', date_to: '', account: '', direction: '' },
 { title: 'PT FAS — Mutasi Kas & Bank Tercatat', filename: 'PTFAS_Kas_Bank.xlsx', headings: ['Waktu WIB', 'Akun', 'Jenis', 'Referensi', 'Pihak', 'Keterangan', 'Arah', 'Nominal', 'Saldo berjalan tercatat'],
 rows: items => items.map(p => [movementDate(p), accountLabel(p.account), kindLabel(p.kind), p.reference_number, p.party, p.description, p.direction, Number(p.amount), Number(p.running_balance)]),
 description: s => `Saldo awal tercatat ${money(s.opening)} · Masuk ${money(s.in_amount)} · Keluar ${money(s.out_amount)} · Saldo akhir tercatat ${money(s.closing)}. Total mencakup seluruh akun/periode terpilih sebelum pencarian/arah.` })
</script>
<template>
 <main class="finance-ledger"><header class="page-heading"><div><p class="eyebrow">Finance · Cash & Bank</p><h1>Kas & Bank</h1><p class="muted">Mutasi otomatis dari pembayaran customer, pengeluaran, refund, dan pembatalan pencatatan.</p></div><div class="actions"><button :disabled="loading || exporting" @click="load()">Perbarui</button><button :disabled="!ready || loading || exporting" @click="report('excel')">Excel</button><button :disabled="!ready || loading || exporting" @click="report('print')">Print / PDF</button></div></header><FinanceNavigation />
 <p class="message">Ini saldo transaksi yang dicatat di sistem, bukan saldo rekening bank sebenarnya. Midtrans ditampilkan terpisah sebelum biaya dan pencairan. Saldo awal rekening, transfer antar rekening, serta rekonsiliasi bank belum dimasukkan.</p>
 <p v-if="error" class="message error" role="alert">{{ error }}</p>
 <section class="cards"><article class="card"><span>Saldo awal tercatat</span><strong>{{ ready ? money(summary.opening) : '—' }}</strong><small>Sebelum tanggal awal terpilih</small></article><article class="card"><span>Mutasi masuk / keluar periode</span><strong>{{ ready ? money(summary.in_amount) : '—' }}</strong><small>Keluar: {{ ready ? money(summary.out_amount) : '—' }}</small></article><article class="card"><span>Saldo akhir tercatat</span><strong>{{ ready ? money(summary.closing) : '—' }}</strong><small>Saldo awal + masuk − keluar</small></article></section>
 <form class="panel" @submit.prevent="load(true)"><fieldset :disabled="loading || exporting"><div class="filters">
 <label>Cari mutasi<input v-model="filters.query" type="search" maxlength="200" placeholder="Customer, penerima, nomor, uraian" /></label><label>Dari tanggal<input v-model="filters.date_from" type="date" /></label><label>Sampai tanggal<input v-model="filters.date_to" type="date" /></label>
 <label>Akun<FasSelect v-model="filters.account" :disabled="loading || exporting" aria-label="Akun kas"><option value="">Semua akun</option><option value="CASH">Tunai</option><option value="BANK">Bank</option><option value="MIDTRANS">Midtrans belum direkonsiliasi</option><option value="LEGACY">Pembayaran lama</option></FasSelect></label>
 <label>Arah<FasSelect v-model="filters.direction" :disabled="loading || exporting" aria-label="Arah mutasi"><option value="">Semua arah</option><option value="IN">Masuk</option><option value="OUT">Keluar</option></FasSelect></label><div class="actions"><button class="primary" type="submit">Terapkan Filter</button></div></div></fieldset></form>
 <section class="panel"><header class="panel-heading"><div><h2>Buku Mutasi</h2><small>{{ description(applied) }}</small><small>Kartu saldo mengikuti akun dan tanggal. Pencarian/arah menyaring baris tanpa mengubah saldo berjalan.</small></div></header>
 <div class="table-scroll"><table><thead><tr><th>Waktu WIB</th><th>Akun / Jenis</th><th>Pihak / Keterangan</th><th class="money">Masuk</th><th class="money">Keluar</th><th class="money">Saldo Tercatat</th><th>Aksi</th></tr></thead><tbody>
 <tr v-for="item in data" :key="item.id"><td>{{ movementDate(item) }}<small>{{ item.reference_number }}</small></td><td>{{ accountLabel(item.account) }}<small>{{ kindLabel(item.kind) }}</small></td><td>{{ item.party }}<small>{{ item.description }}</small></td><td class="money">{{ item.direction === 'IN' ? money(item.amount) : '—' }}</td><td class="money">{{ item.direction === 'OUT' ? money(item.amount) : '—' }}</td><td class="money">{{ money(item.running_balance) }}</td><td><button v-if="item.invoice_id" @click="router.push({ path: '/finance/payments', query: { invoice: item.invoice_id } })">Lihat Pembayaran</button><button v-else @click="router.push('/finance/expenses')">Pengeluaran</button></td></tr>
 <tr v-if="!data.length"><td colspan="7" class="empty">{{ loading ? 'Memuat mutasi…' : 'Belum ada mutasi sesuai filter.' }}</td></tr></tbody></table></div><footer class="pagination"><span>{{ total }} data · Halaman {{ page + 1 }} · {{ loading ? 'Memuat…' : exporting ? 'Menyiapkan seluruh data…' : '' }}</span><div class="actions"><button :disabled="!canPrevious" @click="load(false, page - 1)">Sebelumnya</button><button :disabled="!canNext" @click="load(false, page + 1)">Berikutnya</button></div></footer></section></main>
</template>
