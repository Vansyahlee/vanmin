<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { getFinanceNotifications, readFinanceNotification, readAllFinanceNotifications } from '@/services/finance'

import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import FasSelect from '@/components/ui/FasSelect.vue'
import { useFinanceLedger, financeMoney as money, financeDate as date, financeToday, financeError } from '@/composables/useFinanceLedger'
import '@/assets/finance-ledger.css'

const router = useRouter(), busy = ref(false)
const kindLabel = value => ({ PAYMENT_CONFIRMED: 'Pembayaran terkonfirmasi', PAYMENT_IMPORTED: 'Pembayaran lama diimpor', PAYMENT_REFUND: 'Refund / pengurangan pembayaran', PAYMENT_VOID: 'Pembayaran dibatalkan' })[value] || value
const { filters, applied, data, total, metadata, extra, loading, exporting, ready, error, page, load, report, description, canPrevious, canNext } = useFinanceLedger(getFinanceNotifications,
 { query: '', date_from: '', date_to: '', method: '', read: 'ALL' },
 { title: 'PT FAS — Riwayat Notifikasi Pembayaran', filename: 'PTFAS_Notifikasi_Pembayaran.xlsx', headings: ['Waktu WIB', 'Customer', 'Faktur', 'Peristiwa', 'Nominal peristiwa', 'Sumber', 'Metode', 'Referensi', 'Status baca saya', 'Status pembayaran saat diambil', 'Pembayaran saat ini setelah refund'],
 rows: items => items.map(p => [date(p.created_at), p.customer_name, p.invoice_number, kindLabel(p.kind), Number(p.amount), p.source, p.method, p.reference_number, p.is_read ? 'Dibaca' : 'Belum dibaca', p.payment_status, Number(p.current_payment_net)]),
 description: () => 'Riwayat peristiwa; pembayaran yang kemudian direfund/dibatalkan tetap tercantum. Bukan penjumlahan penerimaan bersih.' })
const announce = () => window.dispatchEvent(new Event('fas-finance-notifications-changed'))
const mark = async item => {
 if (busy.value || exporting.value) return
 busy.value = true
 try { await readFinanceNotification(item.id); await load(); announce() } catch (failure) { error.value = financeError(failure) } finally { busy.value = false }
}
const open = async item => { if (!item.is_read) await mark(item); router.push({ path: '/finance/payments', query: { invoice: item.invoice_id } }) }
const markAll = async () => {
 if (busy.value || exporting.value || !ready.value) return
 busy.value = true
 try { await readAllFinanceNotifications(metadata.value.through_id || '0'); await load(); announce() } catch (failure) { error.value = financeError(failure) } finally { busy.value = false }
}
</script>
<template>
 <main class="finance-ledger"><header class="page-heading"><div><p class="eyebrow">Finance · Notifications</p><h1>Riwayat Pembayaran Masuk</h1><p class="muted">Customer, faktur, cicilan, refund, dan pembatalan. Status dibaca tersimpan khusus akun lu.</p></div><div class="actions"><button :disabled="loading || exporting" @click="load()">Perbarui</button><button :disabled="!ready || loading || exporting" @click="report('excel')">Excel</button><button :disabled="!ready || loading || exporting" @click="report('print')">Print / PDF</button></div></header><FinanceNavigation />
 <p class="message">Notifikasi muncul setelah konfirmasi backend, bukan saat customer baru membuka tautan. Pembayaran manual diberi sumber MANUAL. Riwayat lama yang diimpor dianggap sudah dibaca.</p>
 <p v-if="error" class="message error" role="alert">{{ error }}</p>
 <form class="panel" @submit.prevent="load(true)"><fieldset :disabled="loading || exporting || busy"><div class="filters">
 <label>Cari customer / faktur<input v-model="filters.query" type="search" maxlength="200" placeholder="Nama toko, faktur, referensi" /></label><label>Dari tanggal<input v-model="filters.date_from" type="date" /></label><label>Sampai tanggal<input v-model="filters.date_to" type="date" /></label>
 <label>Sumber<FasSelect v-model="filters.method" :disabled="loading || exporting || busy" aria-label="Sumber pembayaran"><option value="">Semua</option><option value="MIDTRANS">Midtrans</option><option value="MANUAL">Catatan manual</option></FasSelect></label>
 <label>Status baca saya<FasSelect v-model="filters.read" :disabled="loading || exporting || busy" aria-label="Status baca notifikasi"><option value="ALL">Semua</option><option value="UNREAD">Belum dibaca</option><option value="READ">Sudah dibaca</option></FasSelect></label><div class="actions"><button class="primary" type="submit">Terapkan Filter</button></div></div></fieldset></form>
 <section class="panel"><header class="panel-heading"><div><h2>Riwayat · {{ extra.unread_count || 0 }} belum dibaca seluruh periode</h2><small>{{ description(applied) }}</small></div><button :disabled="busy || loading || exporting || !ready" @click="markAll">Tandai Semua Saat Ini Dibaca</button></header>
 <div class="table-scroll"><table><thead><tr><th>Waktu WIB / Baca</th><th>Customer / Faktur</th><th>Peristiwa</th><th class="money">Nominal Peristiwa</th><th>Sumber / Metode</th><th>Status Pembayaran Kini</th><th>Aksi</th></tr></thead><tbody>
 <tr v-for="item in data" :key="item.id" :class="{ unread: !item.is_read }"><td>{{ date(item.created_at) }}<small>{{ item.is_read ? 'Sudah dibaca' : '● Belum dibaca' }}</small></td><td>{{ item.customer_name }}<small>{{ item.invoice_number }}</small></td><td>{{ kindLabel(item.kind) }}<small>{{ item.is_historical ? 'Impor saldo lama setelah refund' : item.reference_number }}</small></td><td class="money">{{ money(item.amount) }}</td><td>{{ item.source }}<small>{{ item.method }}</small></td><td>{{ item.payment_status }}<small>Pembayaran kini: {{ money(item.current_payment_net) }}</small></td><td><div class="actions"><button :disabled="busy || exporting" @click="open(item)">Lihat Pembayaran / Faktur</button><button v-if="!item.is_read" :disabled="busy || exporting" @click="mark(item)">Tandai Dibaca</button></div></td></tr>
 <tr v-if="!data.length"><td colspan="7" class="empty">{{ loading ? 'Memuat notifikasi…' : 'Belum ada notifikasi sesuai filter.' }}</td></tr></tbody></table></div><footer class="pagination"><span>{{ total }} data · Halaman {{ page + 1 }} · {{ loading ? 'Memuat…' : exporting ? 'Menyiapkan seluruh data…' : '' }}</span><div class="actions"><button :disabled="!canPrevious" @click="load(false, page - 1)">Sebelumnya</button><button :disabled="!canNext" @click="load(false, page + 1)">Berikutnya</button></div></footer></section></main>
</template>
