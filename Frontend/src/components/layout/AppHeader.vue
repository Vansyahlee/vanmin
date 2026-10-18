<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'

const emit = defineEmits(['toggle-mobile-sidebar'])
const route = useRoute()

const pageTitles = {
  '/dashboard': 'Overview',
  '/warehouse': 'Dashboard Gudang',
  '/operations': 'Operations',
  '/inventory': 'Inventory',
  '/distribution': 'Distribution',
  '/intelligence': 'Intelligence',
  '/finance': 'Finance Overview',
  '/finance/analytics': 'Finance Analytics',
  '/finance/sales': 'Finance Penjualan',
  '/finance/customers': 'Finance Customer',
  '/finance/receivables': 'Finance Piutang',
  '/finance/payments': 'Finance Pembayaran',
  '/finance/notifications': 'Riwayat Pembayaran Masuk',
  '/finance/expenses': 'Finance Pengeluaran',
  '/finance/cash': 'Finance Kas & Bank',
  '/finance/profit': 'Finance Laba',
  '/finance/tax': 'Finance Pajak',
  '/finance/reports': 'Finance Laporan',
  '/reports': 'Reports',
  '/users': 'Users',
  '/profile': 'Profil Saya',
  '/settings/company': 'Profil Perusahaan',
  '/settings': 'Pengaturan',
}

const pageTitle = computed(() => pageTitles[route.path] || 'PT. FAS')

const currentDate = computed(() => new Intl.DateTimeFormat('id-ID', {
  weekday: 'long',
  day: 'numeric',
  month: 'long',
  year: 'numeric',
}).format(new Date()))

const compactDate = computed(() => new Intl.DateTimeFormat('id-ID', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
}).format(new Date()))
</script>

<template>
  <header
    class="sticky top-0 z-30 flex h-14 items-center justify-between border-b px-4 backdrop-blur-xl sm:h-16 sm:px-6 lg:px-8"
    style="
      border-color: var(--fas-border);
      background: color-mix(in srgb, var(--fas-bg) 92%, transparent);
    "
  >
    <div class="flex min-w-0 items-center gap-3">
      <button
        type="button"
        class="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl border lg:hidden"
        style="
          border-color: var(--fas-border);
          background: var(--fas-overlay);
          color: var(--fas-text-soft);
        "
        aria-label="Buka menu navigasi"
        @click="emit('toggle-mobile-sidebar')"
      >
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" class="h-5 w-5" aria-hidden="true">
          <path d="M4 6h16M4 12h16M4 18h16" />
        </svg>
      </button>

      <h1
        class="truncate text-base font-bold tracking-tight sm:text-lg"
        style="color: var(--fas-text)"
      >
        {{ pageTitle }}
      </h1>
    </div>

    <div
      class="flex shrink-0 items-center gap-2 rounded-xl border px-3 py-2 text-xs sm:px-3.5"
      style="
        border-color: var(--fas-border);
        background: var(--fas-overlay);
        color: var(--fas-text-soft);
      "
      :title="currentDate"
      :aria-label="`Tanggal: ${currentDate}`"
    >
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4" aria-hidden="true">
        <rect x="3" y="5" width="18" height="16" rx="2" />
        <path d="M7 3v4M17 3v4M3 10h18" />
      </svg>
      <span class="hidden font-medium sm:inline">{{ currentDate }}</span>
      <span class="font-medium sm:hidden">{{ compactDate }}</span>
    </div>
  </header>
</template>
