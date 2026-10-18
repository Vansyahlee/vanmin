
<script setup>
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
} from 'vue'

import { RouterLink } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { getProducts } from '@/services/product'
import { getSalesOrders } from '@/services/salesOrder'

// Dashboard gudang: hanya membaca data operasional.
const REFRESH_INTERVAL_MS = 30000

const auth = useAuthStore()

const loading = ref(true)
const refreshing = ref(false)

const products = ref([])
const orders = ref([])

const productsError = ref('')
const ordersError = ref('')

const productsReady = ref(false)
const ordersReady = ref(false)

const updatedAt = ref(null)

let refreshTimer = null
let inFlight = false
let disposed = false

const formatNumber = (value) => {
  return new Intl.NumberFormat('id-ID').format(
    Number(value) || 0,
  )
}

const formatDate = (value) => {
  if (!value) return '—'

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return '—'

  return new Intl.DateTimeFormat('id-ID', {
    day: '2-digit',
    month: 'short',
    year: 'numeric',
  }).format(date)
}

const formatTime = (value) => {
  if (!value) return '—'

  const date = new Date(value)

  if (Number.isNaN(date.getTime())) return '—'

  return new Intl.DateTimeFormat('id-ID', {
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

const normalizedStatus = (order) => {
  return String(order?.status || '')
    .trim()
    .toUpperCase()
}

const statusMeta = {
  DRAFT: {
    label: 'Draft',
    tone: 'muted',
  },
  CONFIRMED: {
    label: 'Menunggu picking',
    tone: 'warning',
  },
  PICKING: {
    label: 'Sedang picking',
    tone: 'info',
  },
  PACKED: {
    label: 'Siap dikirim',
    tone: 'success',
  },
  SHIPPED: {
    label: 'Dikirim',
    tone: 'info',
  },
  COMPLETED: {
    label: 'Selesai',
    tone: 'success',
  },
  CANCELLED: {
    label: 'Dibatalkan',
    tone: 'danger',
  },
}

const statusOf = (order) => {
  return statusMeta[normalizedStatus(order)] || {
    label: normalizedStatus(order) || 'Tidak diketahui',
    tone: 'muted',
  }
}

const activeProducts = computed(() => {
  return products.value.filter(
    (item) => item?.is_active !== false,
  )
})

const lowStockProducts = computed(() => {
  return activeProducts.value
    .filter(
      (item) =>
        Number(item.stock) <=
        Number(item.min_stock ?? 0),
    )
    .sort(
      (a, b) =>
        (Number(a.stock) - Number(a.min_stock ?? 0)) -
        (Number(b.stock) - Number(b.min_stock ?? 0)),
    )
})

const countStatus = (status) => {
  return orders.value.filter(
    (item) => normalizedStatus(item) === status,
  ).length
}

const cards = computed(() => [
  {
    key: 'incoming',
    title: 'Pesanan masuk',
    description: 'Draft dan pesanan terkonfirmasi',
    value: ordersReady.value
      ? formatNumber(
          countStatus('DRAFT') +
          countStatus('CONFIRMED'),
        )
      : '—',
    tone: 'accent',
    icon: 'M8 4h8v4H8z M6 6H4v15h16V6h-2 M8 12h8 M8 16h5',
    route: '/operations',
  },
  {
    key: 'picking',
    title: 'Sedang picking',
    description: 'Barang dalam proses disiapkan',
    value: ordersReady.value
      ? formatNumber(countStatus('PICKING'))
      : '—',
    tone: 'info',
    icon: 'M3 7l9-4 9 4-9 4z M3 7v10l9 4 9-4V7 M12 11v10',
    route: '/operations',
  },
  {
    key: 'packed',
    title: 'Siap dikirim',
    description: 'Pesanan berstatus PACKED',
    value: ordersReady.value
      ? formatNumber(countStatus('PACKED'))
      : '—',
    tone: 'success',
    icon: 'M3 17h2M7 17h10M19 17h2M5 17V7h10v10M15 11h3l3 3v3M8 10h4M8 13h4',
    route: '/distribution',
  },
  {
    key: 'low',
    title: 'Stok menipis',
    description: 'Produk mencapai batas minimum',
    value: productsReady.value
      ? formatNumber(lowStockProducts.value.length)
      : '—',
    tone: 'warning',
    icon: 'M12 3 2 21h20L12 3Zm0 7v4m0 4h.01',
    route: '/inventory',
  },
])

const flow = computed(() => [
  {
    status: 'DRAFT',
    label: 'Draft',
    count: countStatus('DRAFT'),
  },
  {
    status: 'CONFIRMED',
    label: 'Menunggu picking',
    count: countStatus('CONFIRMED'),
  },
  {
    status: 'PICKING',
    label: 'Picking',
    count: countStatus('PICKING'),
  },
  {
    status: 'PACKED',
    label: 'Packed',
    count: countStatus('PACKED'),
  },
])

const pendingTotal = computed(() => {
  return flow.value.reduce(
    (sum, item) => sum + item.count,
    0,
  )
})

const recentOrders = computed(() => {
  return [...orders.value]
    .filter((order) =>
      [
        'DRAFT',
        'CONFIRMED',
        'PICKING',
        'PACKED',
      ].includes(normalizedStatus(order)),
    )
    .sort(
      (a, b) =>
        new Date(b.created_at || b.order_date || 0) -
        new Date(a.created_at || a.order_date || 0),
    )
    .slice(0, 6)
})

const greeting = computed(() => {
  const hour = new Date().getHours()

  if (hour < 11) return 'Selamat pagi'
  if (hour < 15) return 'Selamat siang'
  if (hour < 18) return 'Selamat sore'

  return 'Selamat malam'
})

const refresh = async ({ silent = false } = {}) => {
  if (inFlight || disposed) return

  inFlight = true

  if (!silent) {
    loading.value = true
  } else {
    refreshing.value = true
  }

  const [orderResult, productResult] =
    await Promise.allSettled([
      getSalesOrders(),
      getProducts(),
    ])

  if (disposed) return

  if (
    orderResult.status === 'fulfilled' &&
    Array.isArray(orderResult.value)
  ) {
    orders.value = orderResult.value
    ordersError.value = ''
    ordersReady.value = true
  } else {
    orders.value = []
    ordersReady.value = false

    ordersError.value =
      orderResult.status === 'rejected'
        ? (
            orderResult.reason?.response?.data?.error ||
            'Data pesanan tidak dapat dimuat.'
          )
        : 'Format data pesanan tidak dikenal.'
  }

  if (
    productResult.status === 'fulfilled' &&
    Array.isArray(productResult.value)
  ) {
    products.value = productResult.value
    productsError.value = ''
    productsReady.value = true
  } else {
    products.value = []
    productsReady.value = false

    productsError.value =
      productResult.status === 'rejected'
        ? (
            productResult.reason?.response?.data?.error ||
            'Data produk tidak dapat dimuat.'
          )
        : 'Format data produk tidak dikenal.'
  }

  updatedAt.value = new Date()

  loading.value = false
  refreshing.value = false
  inFlight = false
}

const refreshIfVisible = () => {
  if (document.visibilityState === 'visible') {
    refresh({ silent: true })
  }
}

onMounted(() => {
  refresh()

  refreshTimer = window.setInterval(
    refreshIfVisible,
    REFRESH_INTERVAL_MS,
  )

  document.addEventListener(
    'visibilitychange',
    refreshIfVisible,
  )
})

onBeforeUnmount(() => {
  disposed = true

  window.clearInterval(refreshTimer)

  document.removeEventListener(
    'visibilitychange',
    refreshIfVisible,
  )
})
</script>

<template>
  <div class="warehouse-dashboard">
    <div class="warehouse-inner">

      <header class="warehouse-intro">
        <div>
          <p class="warehouse-eyebrow">
            <span class="warehouse-dot"></span>
            WAREHOUSE WORKSPACE
          </p>

          <h1>
            {{ greeting }},
            {{ auth.user?.name || 'Tim Gudang' }}.
          </h1>

          <p class="warehouse-subtitle">
            Pantau pesanan, persiapan barang,
            pengiriman, dan stok dari satu tempat.
          </p>

          <p class="warehouse-update" aria-live="polite">
            {{
              updatedAt
                ? `Terakhir diperbarui ${formatTime(updatedAt)}`
                : 'Memuat data operasional…'
            }}
            <span class="warehouse-separator">·</span>
            Pembaruan otomatis tiap 30 detik
          </p>
        </div>

        <div class="warehouse-head-actions">
          <button
            class="warehouse-btn warehouse-btn-outline"
            :disabled="loading || refreshing"
            type="button"
            @click="refresh()"
          >
            <svg
              :class="{ spinning: loading || refreshing }"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.7"
            >
              <path
                d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7"
              />
            </svg>

            {{
              loading || refreshing
                ? 'Memperbarui…'
                : 'Perbarui data'
            }}
          </button>

          <RouterLink
            class="warehouse-btn warehouse-btn-primary"
            to="/operations"
          >
            Buka Operations
            <span aria-hidden="true">↗</span>
          </RouterLink>
        </div>
      </header>

      <div
        v-if="ordersError || productsError"
        class="warehouse-errors"
        role="alert"
      >
        <p v-if="ordersError">
          <strong>Pesanan:</strong>
          {{ ordersError }}
        </p>

        <p v-if="productsError">
          <strong>Produk:</strong>
          {{ productsError }}
        </p>
      </div>

      <section
        class="warehouse-metrics"
        aria-label="Ringkasan gudang"
      >
        <RouterLink
          v-for="card in cards"
          :key="card.key"
          :to="card.route"
          class="warehouse-metric"
          :class="`metric-${card.tone}`"
        >
          <div class="warehouse-metric-head">
            <span class="warehouse-metric-label">
              {{ card.title }}
            </span>

            <span class="warehouse-icon">
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.7"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path :d="card.icon" />
              </svg>
            </span>
          </div>

          <strong>{{ card.value }}</strong>

          <div class="warehouse-metric-foot">
            <span>{{ card.description }}</span>
            <span aria-hidden="true">↗</span>
          </div>
        </RouterLink>
      </section>

      <div class="warehouse-main-grid">
        <section class="warehouse-panel warehouse-orders-panel">
          <div class="warehouse-panel-head">
            <div>
              <p class="warehouse-section-label">
                DAFTAR PRIORITAS
              </p>

              <h2>Pesanan yang perlu diproses</h2>

              <p>
                Enam pesanan aktif terbaru berdasarkan
                waktu dibuat.
              </p>
            </div>

            <RouterLink
              class="warehouse-text-link"
              to="/operations"
            >
              Lihat semua
              <span aria-hidden="true">→</span>
            </RouterLink>
          </div>

          <div
            v-if="!ordersReady && loading"
            class="warehouse-placeholder"
          >
            Mengambil data pesanan…
          </div>

          <div
            v-else-if="!ordersReady"
            class="warehouse-placeholder"
          >
            Data pesanan belum dapat ditampilkan.
          </div>

          <div
            v-else-if="recentOrders.length === 0"
            class="warehouse-placeholder"
          >
            Tidak ada pesanan yang sedang menunggu proses.
          </div>

          <div v-else class="warehouse-orders-list">
            <RouterLink
              v-for="order in recentOrders"
              :key="order.id"
              to="/operations"
              class="warehouse-order"
            >
              <div
                class="warehouse-order-symbol"
                aria-hidden="true"
              >
                <svg
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="1.6"
                >
                  <path
                    d="M7 3h10l3 3v15H4V3h3Zm10 0v5h3M8 12h8M8 16h6"
                  />
                </svg>
              </div>

              <div class="warehouse-order-details">
                <strong>
                  {{ order.order_number || 'Sales Order' }}
                </strong>

                <span>
                  {{
                    order.customer_name ||
                    'Customer tidak tercantum'
                  }}
                  <span class="warehouse-separator">·</span>
                  {{
                    formatDate(
                      order.order_date || order.created_at,
                    )
                  }}
                </span>
              </div>

              <span
                class="warehouse-status"
                :class="`status-${statusOf(order).tone}`"
              >
                {{ statusOf(order).label }}
              </span>

              <span
                class="warehouse-order-chevron"
                aria-hidden="true"
              >
                →
              </span>
            </RouterLink>
          </div>
        </section>

        <section class="warehouse-panel warehouse-stock-panel">
          <div class="warehouse-panel-head">
            <div>
              <p class="warehouse-section-label">
                PERSEDIAAN
              </p>

              <h2>Perhatian stok</h2>

              <p>
                Produk aktif dengan stok mencapai
                batas minimum.
              </p>
            </div>

            <RouterLink
              class="warehouse-text-link"
              to="/inventory"
            >
              Inventory
              <span aria-hidden="true">→</span>
            </RouterLink>
          </div>

          <div
            v-if="!productsReady && loading"
            class="warehouse-placeholder"
          >
            Memuat persediaan…
          </div>

          <div
            v-else-if="!productsReady"
            class="warehouse-placeholder"
          >
            Data stok belum dapat ditampilkan.
          </div>

          <div
            v-else-if="lowStockProducts.length === 0"
            class="warehouse-stock-ok"
          >
            <span
              class="warehouse-ok-icon"
              aria-hidden="true"
            >
              ✓
            </span>

            <strong>Stok dalam batas aman</strong>

            <span>
              Tidak ditemukan produk aktif di bawah
              atau sama dengan minimum.
            </span>
          </div>

          <div v-else class="warehouse-lowstock-list">
            <div
              v-for="product in lowStockProducts.slice(0, 5)"
              :key="product.id"
              class="warehouse-lowstock-item"
            >
              <div class="warehouse-product-title">
                <strong>{{ product.name }}</strong>

                <span>
                  {{
                    product.sku ||
                    product.product_code ||
                    'Produk'
                  }}
                  <span class="warehouse-separator">·</span>
                  {{ product.unit || 'unit' }}
                </span>
              </div>

              <div class="warehouse-stock-amount">
                <strong>
                  {{ formatNumber(product.stock) }}
                </strong>

                <span>
                  min.
                  {{ formatNumber(product.min_stock) }}
                </span>
              </div>
            </div>

            <RouterLink
              v-if="lowStockProducts.length > 5"
              to="/inventory"
              class="warehouse-more-link"
            >
              +{{ formatNumber(lowStockProducts.length - 5) }}
              produk lainnya →
            </RouterLink>
          </div>
        </section>
      </div>

      <section class="warehouse-panel warehouse-flow-panel">
        <div class="warehouse-panel-head">
          <div>
            <p class="warehouse-section-label">
              ALUR OPERASIONAL
            </p>

            <h2>Progres pekerjaan gudang</h2>

            <p>
              Ringkasan status seluruh Sales Order
              yang belum dikirim atau diselesaikan.
            </p>
          </div>

          <RouterLink
            class="warehouse-text-link"
            to="/distribution"
          >
            Distribution
            <span aria-hidden="true">→</span>
          </RouterLink>
        </div>

        <div
          v-if="ordersReady"
          class="warehouse-flow-grid"
        >
          <div
            v-for="(step, index) in flow"
            :key="step.status"
            class="warehouse-flow-step"
          >
            <div class="warehouse-flow-top">
              <span class="warehouse-flow-index">
                0{{ index + 1 }}
              </span>

              <span
                v-if="index < flow.length - 1"
                class="warehouse-flow-arrow"
                aria-hidden="true"
              >
                →
              </span>
            </div>

            <strong>
              {{ formatNumber(step.count) }}
            </strong>

            <span>{{ step.label }}</span>

            <div class="warehouse-flow-track">
              <i
                :style="{
                  width: `${
                    pendingTotal
                      ? (step.count / pendingTotal) * 100
                      : 0
                  }%`
                }"
              ></i>
            </div>
          </div>
        </div>

        <div
          v-else
          class="warehouse-placeholder"
        >
          Alur pekerjaan belum dapat dimuat.
        </div>
      </section>

      <section
        class="warehouse-shortcuts"
        aria-label="Akses cepat gudang"
      >
        <RouterLink to="/operations">
          <span>01</span>
          <strong>Kelola pesanan</strong>
          <span aria-hidden="true">↗</span>
        </RouterLink>

        <RouterLink to="/inventory">
          <span>02</span>
          <strong>Cek persediaan</strong>
          <span aria-hidden="true">↗</span>
        </RouterLink>

        <RouterLink to="/distribution">
          <span>03</span>
          <strong>Proses pengiriman</strong>
          <span aria-hidden="true">↗</span>
        </RouterLink>

        <RouterLink to="/intelligence">
          <span>04</span>
          <strong>Rekomendasi distribusi</strong>
          <span aria-hidden="true">↗</span>
        </RouterLink>
      </section>

    </div>
  </div>
</template>

<style scoped>
.warehouse-dashboard {
  min-height: 100%;
  padding: 30px clamp(18px, 3vw, 44px) 55px;
  color: var(--fas-text);
}

.warehouse-inner {
  max-width: 1450px;
  margin: auto;
}

.warehouse-intro {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 22px;
  flex-wrap: wrap;
  margin-bottom: 29px;
}

.warehouse-eyebrow,
.warehouse-section-label {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0 0 13px;
  font-size: 10px;
  font-weight: 750;
  letter-spacing: .19em;
  color: var(--fas-accent-text);
}

.warehouse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--fas-success);
}

.warehouse-intro h1 {
  font-size: clamp(25px, 2.8vw, 39px);
  line-height: 1.2;
  letter-spacing: -.045em;
  font-weight: 780;
  margin: 0;
}

.warehouse-subtitle {
  color: var(--fas-muted);
  font-size: 13px;
  line-height: 1.8;
  margin: 11px 0 0;
  max-width: 700px;
}

.warehouse-update {
  margin: 12px 0 0;
  font-size: 11px;
  color: var(--fas-muted-soft);
}

.warehouse-separator {
  padding: 0 4px;
  color: var(--fas-muted-soft);
}

.warehouse-head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.warehouse-btn {
  display: inline-flex;
  min-height: 43px;
  align-items: center;
  justify-content: center;
  gap: 11px;
  padding: 11px 18px;
  border: 1px solid var(--fas-border-strong);
  border-radius: 13px;
  font-size: 12px;
  font-weight: 730;
  text-decoration: none;
  cursor: pointer;
  transition:
    transform .2s,
    border-color .2s,
    background .2s;
}

.warehouse-btn:hover {
  transform: translateY(-2px);
}

.warehouse-btn:disabled {
  opacity: .65;
  cursor: wait;
}

.warehouse-btn svg {
  width: 15px;
  height: 15px;
}

.warehouse-btn-outline {
  background: var(--fas-card);
  color: var(--fas-text);
}

.warehouse-btn-primary {
  background: var(--fas-accent);
  border-color: var(--fas-accent);
  color: #fff;
}

.spinning {
  animation: fas-warehouse-spin 1s linear infinite;
}

@keyframes fas-warehouse-spin {
  to {
    transform: rotate(360deg);
  }
}

.warehouse-errors {
  border: 1px solid var(--fas-danger);
  background: var(--fas-danger-soft);
  border-radius: 13px;
  padding: 13px 17px;
  color: var(--fas-danger);
  font-size: 12px;
  margin-bottom: 20px;
}

.warehouse-errors p {
  margin: 4px 0;
}

.warehouse-metrics {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 17px;
  margin-bottom: 21px;
}

.warehouse-metric {
  display: flex;
  flex-direction: column;
  min-height: 184px;
  padding: 22px 23px;
  border: 1px solid var(--fas-border);
  border-radius: 23px;
  background: var(--fas-card);
  box-shadow: var(--fas-shadow-soft);
  color: var(--fas-text);
  text-decoration: none;
  transition:
    transform .22s,
    border-color .22s,
    box-shadow .22s;
}

.warehouse-metric:hover {
  transform: translateY(-4px);
  border-color: var(--fas-accent-border);
  box-shadow: var(--fas-shadow-card);
}

.warehouse-metric-head,
.warehouse-metric-foot {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.warehouse-metric-label {
  font-size: 13px;
  font-weight: 750;
  color: var(--fas-text-soft);
}

.warehouse-icon {
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  background: var(--fas-accent-soft);
  border: 1px solid var(--fas-accent-border);
  border-radius: 14px;
  color: var(--fas-accent-text);
}

.warehouse-icon svg {
  width: 22px;
  height: 22px;
}

.metric-success .warehouse-icon {
  background: var(--fas-success-soft);
  border-color: color-mix(
    in srgb,
    var(--fas-success) 25%,
    transparent
  );
  color: var(--fas-success);
}

.metric-warning .warehouse-icon {
  background: var(--fas-warning-soft);
  border-color: color-mix(
    in srgb,
    var(--fas-warning) 25%,
    transparent
  );
  color: var(--fas-warning);
}

.metric-info .warehouse-icon {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.warehouse-metric > strong {
  display: block;
  margin: 15px 0 13px;
  font-size: clamp(26px, 3vw, 39px);
  font-weight: 780;
  letter-spacing: -.045em;
  line-height: 1;
}

.warehouse-metric-foot {
  margin-top: auto;
  color: var(--fas-muted);
  font-size: 10px;
  line-height: 1.5;
}

.warehouse-metric-foot > span:last-child {
  font-size: 16px;
  color: var(--fas-accent-text);
}

.warehouse-main-grid {
  display: grid;
  grid-template-columns:
    minmax(0, 1.55fr)
    minmax(320px, 1fr);
  gap: 20px;
  margin-bottom: 20px;
}

.warehouse-panel {
  min-width: 0;
  background: var(--fas-card);
  border: 1px solid var(--fas-border);
  border-radius: 23px;
  box-shadow: var(--fas-shadow-soft);
}

.warehouse-panel-head {
  padding: 23px 25px 19px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 18px;
  border-bottom: 1px solid var(--fas-border);
}

.warehouse-panel-head h2 {
  font-size: 17px;
  font-weight: 760;
  letter-spacing: -.025em;
  margin: 0;
  color: var(--fas-text);
}

.warehouse-panel-head p:last-child {
  color: var(--fas-muted);
  font-size: 11px;
  line-height: 1.5;
  margin: 8px 0 0;
}

.warehouse-section-label {
  margin-bottom: 9px;
  font-size: 9px;
}

.warehouse-text-link,
.warehouse-more-link {
  white-space: nowrap;
  flex-shrink: 0;
  font-size: 11px;
  font-weight: 740;
  color: var(--fas-accent-text);
  text-decoration: none;
}

.warehouse-text-link:hover,
.warehouse-more-link:hover {
  text-decoration: underline;
}

.warehouse-orders-list {
  padding: 0 8px 8px;
}

.warehouse-order {
  display: flex;
  align-items: center;
  gap: 13px;
  padding: 16px 15px;
  min-width: 0;
  border-bottom: 1px solid var(--fas-border);
  text-decoration: none;
  color: var(--fas-text);
  transition: background .2s;
  border-radius: 9px;
}

.warehouse-order:last-child {
  border-bottom: 0;
}

.warehouse-order:hover {
  background: var(--fas-hover);
}

.warehouse-order-symbol {
  width: 41px;
  height: 41px;
  border: 1px solid var(--fas-border);
  border-radius: 13px;
  flex-shrink: 0;
  background: var(--fas-overlay);
  display: grid;
  place-items: center;
  color: var(--fas-accent-text);
}

.warehouse-order-symbol svg {
  width: 20px;
  height: 20px;
}

.warehouse-order-details {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  flex: 1;
}

.warehouse-order-details strong {
  font-size: 12px;
  font-weight: 740;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.warehouse-order-details > span {
  font-size: 10px;
  color: var(--fas-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.warehouse-status {
  font-size: 10px;
  font-weight: 650;
  padding: 7px 10px;
  border-radius: 999px;
  white-space: nowrap;
  background: var(--fas-overlay);
  color: var(--fas-muted);
}

.status-warning {
  color: var(--fas-warning);
  background: var(--fas-warning-soft);
}

.status-success {
  color: var(--fas-success);
  background: var(--fas-success-soft);
}

.status-info {
  color: var(--fas-accent-text);
  background: var(--fas-accent-soft);
}

.status-danger {
  color: var(--fas-danger);
  background: var(--fas-danger-soft);
}

.warehouse-order-chevron {
  font-size: 17px;
  color: var(--fas-muted);
  padding-left: 4px;
}

.warehouse-placeholder {
  display: grid;
  place-items: center;
  text-align: center;
  padding: 50px 20px;
  min-height: 180px;
  color: var(--fas-muted);
  font-size: 12px;
  line-height: 1.6;
}

.warehouse-stock-ok {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 11px;
  text-align: center;
  min-height: 240px;
  padding: 30px;
}

.warehouse-ok-icon {
  width: 55px;
  height: 55px;
  display: grid;
  place-items: center;
  border-radius: 17px;
  background: var(--fas-success-soft);
  color: var(--fas-success);
  font-size: 23px;
}

.warehouse-stock-ok strong {
  font-size: 13px;
}

.warehouse-stock-ok span:last-child {
  color: var(--fas-muted);
  font-size: 11px;
  max-width: 235px;
  line-height: 1.6;
}

.warehouse-lowstock-list {
  padding: 4px 23px 16px;
}

.warehouse-lowstock-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 15px;
  padding: 17px 0;
  border-bottom: 1px solid var(--fas-border);
}

.warehouse-lowstock-item:last-child {
  border-bottom: 0;
}

.warehouse-product-title {
  display: flex;
  flex-direction: column;
  gap: 7px;
  min-width: 0;
}

.warehouse-product-title strong {
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.warehouse-product-title > span {
  font-size: 10px;
  color: var(--fas-muted);
}

.warehouse-stock-amount {
  text-align: right;
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.warehouse-stock-amount strong {
  color: var(--fas-warning);
  font-size: 16px;
}

.warehouse-stock-amount span {
  font-size: 10px;
  color: var(--fas-muted);
}

.warehouse-more-link {
  display: block;
  padding: 12px 0;
}

.warehouse-flow-panel {
  margin-bottom: 20px;
}

.warehouse-flow-grid {
  padding: 22px 25px 26px;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 15px;
}

.warehouse-flow-step {
  padding: 17px 18px;
  background: var(--fas-elevated);
  border-radius: 17px;
  border: 1px solid var(--fas-border);
  display: flex;
  flex-direction: column;
}

.warehouse-flow-top {
  display: flex;
  justify-content: space-between;
  color: var(--fas-muted);
}

.warehouse-flow-index {
  font-size: 10px;
  font-weight: 740;
}

.warehouse-flow-arrow {
  color: var(--fas-accent-text);
}

.warehouse-flow-step strong {
  font-size: 27px;
  margin: 14px 0 6px;
  font-weight: 780;
}

.warehouse-flow-step > span:not(.warehouse-flow-index) {
  color: var(--fas-muted);
  font-size: 11px;
}

.warehouse-flow-track {
  margin-top: 19px;
  height: 5px;
  border-radius: 5px;
  background: var(--fas-border-strong);
  overflow: hidden;
}

.warehouse-flow-track i {
  display: block;
  height: 100%;
  background: var(--fas-accent);
  border-radius: 5px;
  transition: width .4s ease;
}

.warehouse-shortcuts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 11px;
}

.warehouse-shortcuts a {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 16px;
  border: 1px solid var(--fas-border);
  border-radius: 15px;
  background: var(--fas-card);
  color: var(--fas-text);
  text-decoration: none;
  transition:
    background .2s,
    border-color .2s;
}

.warehouse-shortcuts a:hover {
  background: var(--fas-hover);
  border-color: var(--fas-accent-border);
}

.warehouse-shortcuts a span {
  font-size: 10px;
  color: var(--fas-accent-text);
}

.warehouse-shortcuts a strong {
  font-size: 11px;
  font-weight: 730;
}

.warehouse-dashboard :is(a, button):focus-visible {
  outline: 2px solid var(--fas-accent);
  outline-offset: 3px;
}

@media (max-width: 1250px) {
  .warehouse-metrics {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .warehouse-main-grid {
    grid-template-columns:
      minmax(0, 1.3fr)
      minmax(300px, 1fr);
  }
}

@media (max-width: 900px) {
  .warehouse-main-grid {
    grid-template-columns: 1fr;
  }

  .warehouse-flow-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .warehouse-shortcuts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 580px) {
  .warehouse-dashboard {
    padding: 22px 15px 35px;
  }

  .warehouse-head-actions {
    width: 100%;
  }

  .warehouse-btn {
    flex: 1;
  }

  .warehouse-metrics {
    gap: 10px;
  }

  .warehouse-metric {
    min-height: 163px;
    padding: 17px 15px;
    border-radius: 17px;
  }

  .warehouse-metric-head {
    align-items: flex-start;
  }

  .warehouse-icon {
    width: 36px;
    height: 36px;
    flex-shrink: 0;
  }

  .warehouse-icon svg {
    width: 18px;
    height: 18px;
  }

  .warehouse-metric-label {
    font-size: 11px;
  }

  .warehouse-metric-foot {
    font-size: 9px;
  }

  .warehouse-panel {
    border-radius: 17px;
  }

  .warehouse-panel-head {
    padding: 20px 17px;
    align-items: flex-start;
  }

  .warehouse-orders-list {
    padding: 0 5px 6px;
  }

  .warehouse-order {
    gap: 8px;
    padding: 13px 9px;
  }

  .warehouse-order-symbol {
    display: none;
  }

  .warehouse-status {
    font-size: 9px;
    padding: 6px;
  }

  .warehouse-order-chevron {
    display: none;
  }

  .warehouse-flow-grid {
    padding: 14px;
    gap: 8px;
  }

  .warehouse-flow-step {
    padding: 15px 12px;
  }

  .warehouse-shortcuts {
    gap: 8px;
  }

  .warehouse-shortcuts a {
    padding: 13px;
  }

  .warehouse-shortcuts a strong {
    font-size: 10px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .warehouse-dashboard * {
    transition: none !important;
    animation: none !important;
  }
}
</style>
