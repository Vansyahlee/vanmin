<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useModalDialog } from '@/composables/useModalDialog'
import { getFinanceNotifications, readFinanceNotification } from '@/services/finance'
import { financeMoney as money, financeDate as date, financeError } from '@/composables/useFinanceLedger'
const router = useRouter(), auth = useAuthStore()
const allowed = computed(() => ['admin', 'finance'].includes(auth.user?.role))
const open = ref(false), items = ref([]), unread = ref(0), error = ref(''), loading = ref(false)
let timer, disposed = false, sequence = 0
const close = () => { open.value = false }
useModalDialog(() => open.value && allowed.value ? '#finance-notification-dialog' : '', close)
const refresh = async () => {
 if (!allowed.value || loading.value || disposed || document.visibilityState === 'hidden') return
 const seq = ++sequence, actor = auth.user?.id
 loading.value = true
 try {
  const response = await getFinanceNotifications({ limit: 6, offset: 0, read: 'ALL' })
  if (disposed || seq !== sequence || !allowed.value || auth.user?.id !== actor) return
  items.value = response.data || []; unread.value = Number(response.unread_count || 0); error.value = ''
 } catch (failure) { if (!disposed && seq === sequence && allowed.value) error.value = financeError(failure) }
 finally { if (!disposed && seq === sequence) loading.value = false }
}
const toggle = () => { open.value = !open.value; if (open.value) refresh() }
const kindLabel = kind => ({ PAYMENT_CONFIRMED: 'Pembayaran terkonfirmasi', PAYMENT_IMPORTED: 'Pembayaran lama', PAYMENT_REFUND: 'Refund / pengurangan', PAYMENT_VOID: 'Pembayaran dibatalkan' })[kind] || kind
const showInvoice = async item => {
 try {
  if (!item.is_read) await readFinanceNotification(item.id)
  close(); router.push({ path: '/finance/payments', query: { invoice: item.invoice_id } }); refresh()
 } catch (failure) { error.value = financeError(failure) }
}
const showHistory = () => { close(); router.push('/finance/notifications') }
watch(() => `${auth.user?.id || ''}:${auth.user?.role || ''}`, () => {
 sequence++; clearInterval(timer); items.value = []; unread.value = 0; loading.value = false; error.value = ''; close()
 if (allowed.value) { refresh(); timer = setInterval(refresh, 30000) }
}, { immediate: true })
window.addEventListener('focus', refresh)
document.addEventListener('visibilitychange', refresh)
window.addEventListener('fas-finance-notifications-changed', refresh)
onBeforeUnmount(() => {
 disposed = true; sequence++; clearInterval(timer)
 window.removeEventListener('focus', refresh); document.removeEventListener('visibilitychange', refresh)
 window.removeEventListener('fas-finance-notifications-changed', refresh)
})
</script>
<template>
 <button v-if="allowed" class="finance-bell" type="button" :aria-label="`Notifikasi pembayaran, ${unread} belum dibaca`" aria-haspopup="dialog" :aria-expanded="open" @click="toggle">
 <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 8a6 6 0 1 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4" /></svg><span v-if="unread" class="count">{{ unread > 99 ? '99+' : unread }}</span></button>
 <Teleport to="body"><div v-if="open && allowed" class="finance-notification-backdrop" @click.self="close"><section id="finance-notification-dialog" class="notification-dialog" role="dialog" aria-modal="true" aria-labelledby="notification-dialog-title">
 <header><div><h2 id="notification-dialog-title">Pembayaran Customer</h2><p>{{ unread }} belum dibaca · diperbarui tiap 30 detik saat tab aktif</p></div><button type="button" aria-label="Tutup notifikasi" @click="close">Tutup</button></header>
 <div class="notification-content"><p v-if="error" class="error" role="alert">{{ error }}</p><p v-if="loading" role="status">Memperbarui…</p><p v-if="!loading && !items.length && !error">Belum ada riwayat pembayaran masuk.</p>
 <button v-for="item in items" :key="item.id" class="notification-item" :class="{ unread: !item.is_read }" @click="showInvoice(item)"><span class="row"><strong>{{ item.customer_name || 'Customer' }}</strong><b>{{ money(item.amount) }}</b></span><span>{{ kindLabel(item.kind) }} · {{ item.invoice_number }}</span><small>{{ date(item.created_at) }} · {{ item.source }} · {{ item.is_read ? 'Dibaca' : 'Belum dibaca' }}</small><small>Status pembayaran kini: {{ item.payment_status }}</small></button></div>
 <footer><button type="button" :disabled="loading" @click="refresh">Perbarui</button><button type="button" class="accent" @click="showHistory">Lihat Semua Riwayat</button></footer>
 </section></div></Teleport>
</template>
<style scoped>
.finance-bell{position:relative;display:grid;place-items:center;width:2.85rem;height:2.85rem;border:1px solid var(--fas-border);border-radius:.85rem;background:var(--fas-overlay);color:var(--fas-muted);cursor:pointer}.finance-bell svg{width:1.25rem;height:1.25rem}.count{position:absolute;right:-.35rem;top:-.35rem;min-width:1.2rem;height:1.2rem;border-radius:1rem;background:var(--fas-accent);color:white;font-size:.65rem;font-weight:750;display:grid;place-items:center;padding:0 .2rem}.finance-notification-backdrop{position:fixed;inset:0;z-index:1250;background:rgba(2,6,23,.35);display:flex;align-items:flex-start;justify-content:flex-end;padding:6.5rem 1.25rem 1rem}.notification-dialog{width:min(100%,30rem);max-height:calc(100dvh - 7.5rem);overflow:auto;border:1px solid var(--fas-border-strong);border-radius:1.1rem;background:var(--fas-elevated);box-shadow:var(--fas-shadow-card);color:var(--fas-text);font-size:.8rem}.notification-dialog header,.notification-dialog footer{display:flex;align-items:center;justify-content:space-between;gap:.7rem;padding:1rem;background:var(--fas-elevated)}.notification-dialog header{border-bottom:1px solid var(--fas-border);position:sticky;top:0;z-index:1}.notification-dialog footer{border-top:1px solid var(--fas-border)}h2{font-size:1rem;font-weight:750;margin:0}p{font-size:.75rem;color:var(--fas-muted);line-height:1.6;margin:.3rem 0 0}.notification-dialog button{min-height:2.65rem;font:inherit;border:1px solid var(--fas-border);border-radius:.75rem;background:var(--fas-card);color:var(--fas-text);padding:.6rem .8rem;cursor:pointer}.notification-content{padding:.65rem}.notification-item{display:flex;width:100%;flex-direction:column;align-items:stretch;gap:.4rem;text-align:left;margin-bottom:.5rem}.notification-item.unread{border-color:var(--fas-accent-border);background:var(--fas-accent-soft)}.row{display:flex;justify-content:space-between;flex-wrap:wrap;gap:.5rem}small{color:var(--fas-muted);font-size:.7rem;line-height:1.5}.notification-dialog .accent{background:var(--fas-accent);color:white;border-color:var(--fas-accent)}.notification-dialog .error{color:var(--fas-danger);padding:.7rem;background:var(--fas-danger-soft);border-radius:.75rem}button:focus-visible{outline:2px solid var(--fas-accent);outline-offset:2px}button:disabled{opacity:.5;cursor:not-allowed}@media(max-width:600px){.finance-notification-backdrop{padding:5.25rem .5rem .5rem}.notification-dialog{max-height:calc(100dvh - 5.75rem)}}
</style>
