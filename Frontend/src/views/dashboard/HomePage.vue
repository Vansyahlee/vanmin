<script setup>
import {
  computed,
  onBeforeUnmount,
  onMounted,
  ref,
} from 'vue'

import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { getProducts } from '@/services/product'
import { getSalesOrders } from '@/services/salesOrder'
import {
  getFinanceSalesSummary,
  getFinanceSalesTrend,
} from '@/services/finance'

const AUTO_REFRESH_MS = 15000

const router = useRouter()
const authStore = useAuthStore()

const loading = ref(false)
const refreshing = ref(false)
const errorMessage = ref('')
const lastUpdatedAt = ref(null)

const products = ref([])
const orders = ref([])
const summary = ref({})
const trendRows = ref([])
const granularity = ref('day')

const selectedMetric = ref('grand_total')
const hoveredIndex = ref(null)

let refreshTimer = null

const chartWidth = 1000
const chartHeight = 320

const chartPadding = {
  left: 74,
  right: 24,
  top: 24,
  bottom: 52,
}

const chartTop = chartPadding.top
const chartBottom = chartHeight - chartPadding.bottom
const chartPlotWidth =
  chartWidth -
  chartPadding.left -
  chartPadding.right
const chartPlotHeight =
  chartBottom -
  chartTop

const metricOptions = [
  {
    key: 'grand_total',
    label: 'Penjualan',
    kind: 'currency',
  },
  {
    key: 'net_sales',
    label: 'Bersih',
    kind: 'currency',
  },
  {
    key: 'invoice_count',
    label: 'Faktur',
    kind: 'number',
  },
]

const toISODate = value => {
  const date = new Date(value)

  const year = date.getFullYear()
  const month = String(
    date.getMonth() + 1,
  ).padStart(2, '0')
  const day = String(
    date.getDate(),
  ).padStart(2, '0')

  return `${year}-${month}-${day}`
}

const todayISO = () => {
  return toISODate(
    new Date(),
  )
}

const firstDayOfMonthISO = () => {
  const now = new Date()

  return toISODate(
    new Date(
      now.getFullYear(),
      now.getMonth(),
      1,
    ),
  )
}

const formatNumber = value => {
  return new Intl.NumberFormat(
    'id-ID',
    {
      maximumFractionDigits: 0,
    },
  ).format(
    Number(value || 0),
  )
}

const formatCurrency = value => {
  return new Intl.NumberFormat(
    'id-ID',
    {
      style: 'currency',
      currency: 'IDR',
      maximumFractionDigits: 0,
    },
  ).format(
    Number(value || 0),
  )
}

const formatCompactCurrency = value => {
  const amount = Number(
    value || 0,
  )

  if (
    Math.abs(amount) >=
    1_000_000_000
  ) {
    return `Rp${(
      amount /
      1_000_000_000
    ).toLocaleString(
      'id-ID',
      {
        maximumFractionDigits: 1,
      },
    )} M`
  }

  if (
    Math.abs(amount) >=
    1_000_000
  ) {
    return `Rp${(
      amount /
      1_000_000
    ).toLocaleString(
      'id-ID',
      {
        maximumFractionDigits: 1,
      },
    )} Jt`
  }

  if (
    Math.abs(amount) >=
    1_000
  ) {
    return `Rp${(
      amount /
      1_000
    ).toLocaleString(
      'id-ID',
      {
        maximumFractionDigits: 1,
      },
    )} Rb`
  }

  return formatCurrency(
    amount,
  )
}

const formatDateTime = value => {
  if (!value) {
    return '—'
  }

  return new Intl.DateTimeFormat(
    'id-ID',
    {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    },
  ).format(
    new Date(value),
  )
}

const formatPeriod = value => {
  if (!value) {
    return '—'
  }

  const date = new Date(
    `${String(value).slice(0, 10)}T00:00:00`,
  )

  if (
    granularity.value ===
    'month'
  ) {
    return new Intl.DateTimeFormat(
      'id-ID',
      {
        month: 'short',
        year: '2-digit',
      },
    ).format(date)
  }

  return new Intl.DateTimeFormat(
    'id-ID',
    {
      day: '2-digit',
      month: 'short',
    },
  ).format(date)
}

const currentMetric = computed(() => {
  return (
    metricOptions.find(
      item =>
        item.key ===
        selectedMetric.value,
    ) ||
    metricOptions[0]
  )
})

const userDisplayName = computed(() => {
  const name = String(
    authStore.user?.name ||
    authStore.user?.full_name ||
    '',
  ).trim()

  if (name) {
    return name
  }

  return 'PT FAS Admin'
})

const userRoleLabel = computed(() => {
  const role = String(
    authStore.user?.role ||
    'administrator',
  )
    .trim()
    .toLowerCase()

  if (
    role === 'admin' ||
    role === 'administrator' ||
    role === 'owner'
  ) {
    return 'Administrator'
  }

  if (
    role === 'pegawai' ||
    role === 'warehouse' ||
    role === 'gudang'
  ) {
    return 'Pegawai Gudang'
  }

  if (role === 'finance') {
    return 'Finance'
  }

  return role || 'Administrator'
})

const activeProducts = computed(() => {
  return products.value.filter(
    product =>
      product.is_active !== false,
  )
})

const activeProductCount = computed(() => {
  return activeProducts.value.length
})

const warehouseStock = computed(() => {
  return activeProducts.value.reduce(
    (total, product) =>
      total +
      Number(
        product.stock ||
        0,
      ),
    0,
  )
})

const distributionActivity = computed(() => {
  return orders.value.filter(
    order =>
      [
        'PACKED',
        'SHIPPED',
        'COMPLETED',
      ].includes(
        String(
          order.status ||
          '',
        ).toUpperCase(),
      ),
  ).length
})

const revenue = computed(() => {
  return Number(
    summary.value?.grand_total ||
    0,
  )
})

const chartData = computed(() => {
  return trendRows.value.map(
    row => ({
      ...row,
      metricValue: Number(
        row[
          selectedMetric.value
        ] ||
        0,
      ),
    }),
  )
})

const chartMax = computed(() => {
  const values =
    chartData.value.map(
      item =>
        item.metricValue,
    )

  const max = Math.max(
    0,
    ...values,
  )

  if (max <= 0) {
    return 1
  }

  return max * 1.08
})

const chartMaxInvoices = computed(() => {
  const max = Math.max(
    0,
    ...chartData.value.map(
      item =>
        Number(
          item.invoice_count ||
          0,
        ),
    ),
  )

  return Math.max(
    max,
    1,
  )
})

const pointX = index => {
  const total =
    chartData.value.length

  if (total <= 1) {
    return (
      chartPadding.left +
      chartPlotWidth / 2
    )
  }

  return (
    chartPadding.left +
    (
      index /
      (total - 1)
    ) *
      chartPlotWidth
  )
}

const pointY = value => {
  const ratio =
    Number(value || 0) /
    chartMax.value

  return (
    chartBottom -
    ratio *
      chartPlotHeight
  )
}

const chartPoints = computed(() => {
  return chartData.value.map(
    (
      item,
      index,
    ) => ({
      ...item,
      x: pointX(index),
      y: pointY(
        item.metricValue,
      ),
    }),
  )
})

const chartLinePath = computed(() => {
  if (
    chartPoints.value.length ===
    0
  ) {
    return ''
  }

  return chartPoints.value
    .map(
      (
        point,
        index,
      ) =>
        `${
          index === 0
            ? 'M'
            : 'L'
        } ${point.x.toFixed(
          2,
        )} ${point.y.toFixed(
          2,
        )}`,
    )
    .join(' ')
})

const chartAreaPath = computed(() => {
  const points =
    chartPoints.value

  if (
    points.length ===
    0
  ) {
    return ''
  }

  const first = points[0]
  const last =
    points[
      points.length - 1
    ]

  return (
    `${chartLinePath.value} ` +
    `L ${last.x.toFixed(
      2,
    )} ${chartBottom} ` +
    `L ${first.x.toFixed(
      2,
    )} ${chartBottom} Z`
  )
})

const yTicks = computed(() => {
  return [
    1,
    0.75,
    0.5,
    0.25,
    0,
  ].map(
    ratio => ({
      ratio,
      y:
        chartTop +
        (1 - ratio) *
          chartPlotHeight,
      value:
        chartMax.value *
        ratio,
    }),
  )
})

const xLabels = computed(() => {
  const data =
    chartData.value

  if (
    data.length ===
    0
  ) {
    return []
  }

  const desired = Math.min(
    6,
    data.length,
  )

  if (desired === 1) {
    return [
      {
        index: 0,
        x: pointX(0),
        label: formatPeriod(
          data[0].period_start,
        ),
      },
    ]
  }

  const indexes =
    new Set()

  for (
    let position = 0;
    position < desired;
    position += 1
  ) {
    indexes.add(
      Math.round(
        (
          position /
          (desired - 1)
        ) *
          (data.length - 1),
      ),
    )
  }

  return Array.from(
    indexes,
  ).map(
    index => ({
      index,
      x: pointX(index),
      label: formatPeriod(
        data[index]
          .period_start,
      ),
    }),
  )
})

const hoveredPoint = computed(() => {
  if (
    hoveredIndex.value ===
      null ||
    !chartPoints.value[
      hoveredIndex.value
    ]
  ) {
    return null
  }

  return chartPoints.value[
    hoveredIndex.value
  ]
})

const chartChange = computed(() => {
  if (
    chartData.value.length <
    2
  ) {
    return {
      amount: 0,
      percent: 0,
      direction: 'flat',
      hasPercent: true,
    }
  }

  const first = Number(
    chartData.value[0]
      .metricValue ||
    0,
  )

  const last = Number(
    chartData.value[
      chartData.value.length - 1
    ].metricValue ||
    0,
  )

  const amount =
    last - first

  let percent = 0
  let hasPercent = true

  if (
    first === 0 &&
    last !== 0
  ) {
    hasPercent = false
  } else if (first !== 0) {
    percent =
      (
        amount /
        Math.abs(first)
      ) *
      100
  }

  return {
    amount,
    percent,
    hasPercent,
    direction:
      amount > 0
        ? 'up'
        : amount < 0
          ? 'down'
          : 'flat',
  }
})

const currentMetricValue = computed(() => {
  if (
    chartData.value.length ===
    0
  ) {
    return 0
  }

  return Number(
    chartData.value[
      chartData.value.length - 1
    ].metricValue ||
    0,
  )
})

const currentMetricFormatted = computed(() => {
  if (
    currentMetric.value.kind ===
    'number'
  ) {
    return formatNumber(
      currentMetricValue.value,
    )
  }

  return formatCurrency(
    currentMetricValue.value,
  )
})

const formatChartValue = value => {
  if (
    currentMetric.value.kind ===
    'number'
  ) {
    return formatNumber(
      value,
    )
  }

  return formatCompactCurrency(
    value,
  )
}

const periodLabel = computed(() => {
  const now = new Date()

  return new Intl.DateTimeFormat(
    'id-ID',
    {
      month: 'long',
      year: 'numeric',
    },
  ).format(now)
})

const loadDashboard = async ({
  silent = false,
} = {}) => {
  if (!silent) {
    loading.value = true
  } else {
    refreshing.value = true
  }

  errorMessage.value = ''

  try {
    const params = {
      date_from:
        firstDayOfMonthISO(),
      date_to:
        todayISO(),
      print_state: 'all',
    }

    const [
      productRows,
      orderRows,
      summaryResponse,
      trendResponse,
    ] = await Promise.all([
      getProducts(),
      getSalesOrders(),
      getFinanceSalesSummary(
        params,
      ),
      getFinanceSalesTrend(
        params,
      ),
    ])

    products.value =
      Array.isArray(productRows)
        ? productRows
        : []

    orders.value =
      Array.isArray(orderRows)
        ? orderRows
        : []

    summary.value =
      summaryResponse?.data ||
      {}

    trendRows.value =
      Array.isArray(
        trendResponse?.data,
      )
        ? trendResponse.data
        : []

    granularity.value =
      trendResponse
        ?.granularity ||
      'day'

    lastUpdatedAt.value =
      new Date()

    if (
      hoveredIndex.value !==
        null &&
      hoveredIndex.value >=
        trendRows.value.length
    ) {
      hoveredIndex.value =
        null
    }
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      error.message ||
      'Overview belum dapat dimuat.'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

const selectMetric = key => {
  selectedMetric.value = key
  hoveredIndex.value = null
}

const handleChartPointerMove = event => {
  const total =
    chartPoints.value.length

  if (total <= 0) {
    hoveredIndex.value = null
    return
  }

  const rect =
    event.currentTarget
      .getBoundingClientRect()

  if (rect.width <= 0) {
    return
  }

  const viewX =
    (
      (
        event.clientX -
        rect.left
      ) /
      rect.width
    ) *
    chartWidth

  if (total === 1) {
    hoveredIndex.value = 0
    return
  }

  const normalized =
    (
      viewX -
      chartPadding.left
    ) /
    chartPlotWidth

  const index = Math.round(
    normalized *
      (total - 1),
  )

  hoveredIndex.value =
    Math.max(
      0,
      Math.min(
        total - 1,
        index,
      ),
    )
}

const clearChartHover = () => {
  hoveredIndex.value = null
}

const goQuickAction = () => {
  router.push(
    '/operations',
  )
}

const goFinanceAnalytics = () => {
  router.push(
    '/finance/analytics',
  )
}

onMounted(async () => {
  await loadDashboard()

  refreshTimer =
    window.setInterval(
      () => {
        loadDashboard({
          silent: true,
        })
      },
      AUTO_REFRESH_MS,
    )
})

onBeforeUnmount(() => {
  if (refreshTimer) {
    window.clearInterval(
      refreshTimer,
    )
  }
})
</script>

<template>
  <div class="dashboard-overview">
    <section class="welcome-section">
      <div class="welcome-copy">
        <p class="welcome-kicker">
          Welcome back,
        </p>

        <h1>
          {{ userDisplayName }}
        </h1>

        <div class="role-line">
          <span></span>
          {{ userRoleLabel }}
        </div>

        <p class="welcome-description">
          Monitor company operations, distribution, financial performance and business intelligence from one centralized platform.
        </p>
      </div>

      <button
        type="button"
        class="quick-action-button"
        @click="goQuickAction"
      >
        <span>Quick Action</span>
        <span aria-hidden="true">→</span>
      </button>
    </section>

    <div
      v-if="errorMessage"
      class="overview-message overview-error"
    >
      <span>
        {{ errorMessage }}
      </span>

      <button
        type="button"
        :disabled="loading || refreshing"
        @click="loadDashboard()"
      >
        Coba Lagi
      </button>
    </div>

    <section class="summary-grid">
      <article class="summary-card">
        <div class="summary-card-head">
          <p>Active Products</p>

          <span class="summary-icon">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
            >
              <path d="m12 3 8 4.5-8 4.5-8-4.5L12 3Z" />
              <path d="m4 7.5 8 4.5 8-4.5" />
              <path d="M4 7.5V16l8 5 8-5V7.5" />
            </svg>
          </span>
        </div>

        <strong class="summary-value">
          {{
            loading
              ? '—'
              : formatNumber(
                  activeProductCount,
                )
          }}
        </strong>

        <span class="summary-helper">
          Inventory database
        </span>
      </article>

      <article class="summary-card">
        <div class="summary-card-head">
          <p>Warehouse Stock</p>

          <span class="summary-icon">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
            >
              <path d="M3 10 12 4l9 6" />
              <path d="M5 9.5V20h14V9.5" />
              <path d="M9 20v-6h6v6" />
            </svg>
          </span>
        </div>

        <strong class="summary-value">
          {{
            loading
              ? '—'
              : formatNumber(
                  warehouseStock,
                )
          }}
        </strong>

        <span class="summary-helper">
          Real-time inventory
        </span>
      </article>

      <article class="summary-card">
        <div class="summary-card-head">
          <p>Distribution</p>

          <span class="summary-icon">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
            >
              <path d="M3 7h11v10H3z" />
              <path d="M14 10h4l3 3v4h-7z" />
              <circle cx="7" cy="18" r="2" />
              <circle cx="18" cy="18" r="2" />
            </svg>
          </span>
        </div>

        <strong class="summary-value">
          {{
            loading
              ? '—'
              : formatNumber(
                  distributionActivity,
                )
          }}
        </strong>

        <span class="summary-helper">
          Distribution activity
        </span>
      </article>

      <article class="summary-card">
        <div class="summary-card-head">
          <p>Revenue</p>

          <span class="summary-icon">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
            >
              <path d="M4 19V5" />
              <path d="M4 19h16" />
              <path d="m7 15 4-4 3 2 5-6" />
            </svg>
          </span>
        </div>

        <strong class="summary-value summary-value-money">
          {{
            loading
              ? '—'
              : formatCompactCurrency(
                  revenue,
                )
          }}
        </strong>

        <span class="summary-helper">
          Financial overview
        </span>
      </article>
    </section>

    <section class="performance-panel">
      <header class="performance-header">
        <div>
          <p class="performance-kicker">
            Business Performance
          </p>

          <h2>
            Operational overview
          </h2>
        </div>

        <div class="performance-header-actions">
          <span class="period-chip">
            {{ periodLabel }}
          </span>

          <button
            type="button"
            class="finance-link-button"
            @click="goFinanceAnalytics"
          >
            Finance Analytics
            <span>→</span>
          </button>
        </div>
      </header>

      <div class="chart-toolbar">
        <div class="metric-tabs">
          <button
            v-for="metric in metricOptions"
            :key="metric.key"
            type="button"
            :class="[
              'metric-tab',
              selectedMetric === metric.key
                ? 'metric-tab-active'
                : '',
            ]"
            @click="selectMetric(metric.key)"
          >
            {{ metric.label }}
          </button>
        </div>

        <div class="chart-live-meta">
          <span class="live-pill">
            <i></i>
            LIVE
          </span>

          <span>
            {{ trendRows.length }} titik
          </span>

          <span v-if="lastUpdatedAt">
            {{ formatDateTime(lastUpdatedAt) }}
          </span>

          <button
            type="button"
            class="refresh-chart-button"
            :disabled="loading || refreshing"
            @click="loadDashboard({ silent: true })"
          >
            {{ refreshing ? 'Memuat...' : 'Refresh' }}
          </button>
        </div>
      </div>

      <div class="chart-summary-row">
        <div>
          <span class="chart-summary-label">
            {{ currentMetric.label }} terbaru
          </span>

          <strong>
            {{ currentMetricFormatted }}
          </strong>
        </div>

        <span
          :class="[
            'change-badge',
            `change-${chartChange.direction}`,
          ]"
        >
          <template v-if="chartChange.direction === 'up'">
            ▲
          </template>

          <template v-else-if="chartChange.direction === 'down'">
            ▼
          </template>

          <template v-else>
            ●
          </template>

          {{
            chartChange.hasPercent
              ? `${Math.abs(
                  chartChange.percent,
                ).toLocaleString(
                  'id-ID',
                  {
                    maximumFractionDigits: 1,
                  },
                )}%`
              : 'Baru'
          }}
        </span>
      </div>

      <div
        v-if="loading && trendRows.length === 0"
        class="chart-state"
      >
        <span class="overview-spinner"></span>
        <p>Menyiapkan analytics...</p>
      </div>

      <div
        v-else-if="trendRows.length === 0"
        class="chart-state"
      >
        <span class="chart-empty-icon">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path d="M4 19V5" />
            <path d="M4 19h16" />
            <path d="m7 15 4-4 3 2 5-6" />
          </svg>
        </span>

        <strong>
          Belum ada data analytics
        </strong>

        <p>
          Grafik akan terisi otomatis setelah faktur pada periode aktif tersedia.
        </p>
      </div>

      <div
        v-else
        class="chart-shell"
      >
        <svg
          class="overview-chart"
          :viewBox="`0 0 ${chartWidth} ${chartHeight}`"
          preserveAspectRatio="none"
          @pointermove="handleChartPointerMove"
          @pointerleave="clearChartHover"
        >
          <defs>
            <linearGradient
              id="overview-area-gradient"
              x1="0"
              y1="0"
              x2="0"
              y2="1"
            >
              <stop
                offset="0%"
                stop-color="var(--fas-accent)"
                stop-opacity="0.32"
              />

              <stop
                offset="100%"
                stop-color="var(--fas-accent)"
                stop-opacity="0.015"
              />
            </linearGradient>

            <filter
              id="overview-line-glow"
              x="-20%"
              y="-20%"
              width="140%"
              height="140%"
            >
              <feGaussianBlur
                stdDeviation="2"
                result="blur"
              />

              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>

          <g class="chart-grid-lines">
            <g
              v-for="tick in yTicks"
              :key="tick.ratio"
            >
              <line
                :x1="chartPadding.left"
                :x2="chartWidth - chartPadding.right"
                :y1="tick.y"
                :y2="tick.y"
              />

              <text
                x="8"
                :y="tick.y + 4"
              >
                {{ formatChartValue(tick.value) }}
              </text>
            </g>
          </g>

          <g class="chart-volume-bars">
            <rect
              v-for="(point, index) in chartPoints"
              :key="`bar-${index}`"
              :x="
                point.x -
                Math.max(
                  1.5,
                  Math.min(
                    7,
                    chartPlotWidth /
                      Math.max(
                        chartPoints.length,
                        1,
                      ) /
                      3,
                  ),
                )
              "
              :y="
                chartBottom -
                (
                  Number(
                    point.invoice_count || 0,
                  ) /
                  chartMaxInvoices
                ) *
                  30
              "
              :width="
                Math.max(
                  3,
                  Math.min(
                    14,
                    chartPlotWidth /
                      Math.max(
                        chartPoints.length,
                        1,
                      ) /
                      1.5,
                  ),
                )
              "
              :height="
                (
                  Number(
                    point.invoice_count || 0,
                  ) /
                  chartMaxInvoices
                ) *
                  30
              "
              rx="2"
            />
          </g>

          <path
            class="chart-area-path"
            :d="chartAreaPath"
          />

          <path
            class="chart-line-path"
            :d="chartLinePath"
            filter="url(#overview-line-glow)"
          />

          <g class="chart-point-layer">
            <circle
              v-for="(point, index) in chartPoints"
              :key="`point-${index}`"
              :cx="point.x"
              :cy="point.y"
              :r="hoveredIndex === index ? 5 : 2.8"
              :class="{
                hovered:
                  hoveredIndex === index,
              }"
            />
          </g>

          <g
            v-if="hoveredPoint"
            class="chart-crosshair"
          >
            <line
              :x1="hoveredPoint.x"
              :x2="hoveredPoint.x"
              :y1="chartTop"
              :y2="chartBottom"
            />

            <line
              :x1="chartPadding.left"
              :x2="chartWidth - chartPadding.right"
              :y1="hoveredPoint.y"
              :y2="hoveredPoint.y"
            />

            <circle
              :cx="hoveredPoint.x"
              :cy="hoveredPoint.y"
              r="6.5"
            />
          </g>

          <g class="chart-x-labels">
            <text
              v-for="label in xLabels"
              :key="label.index"
              :x="label.x"
              :y="chartHeight - 18"
              text-anchor="middle"
            >
              {{ label.label }}
            </text>
          </g>

          <rect
            :x="chartPadding.left"
            :y="chartTop"
            :width="chartPlotWidth"
            :height="chartPlotHeight"
            fill="transparent"
          />
        </svg>

        <div
          v-if="hoveredPoint"
          class="chart-tooltip"
          :style="{
            left:
              `${Math.min(
                88,
                Math.max(
                  10,
                  (
                    hoveredPoint.x /
                    chartWidth
                  ) *
                    100,
                ),
              )}%`,
          }"
        >
          <span>
            {{ formatPeriod(hoveredPoint.period_start) }}
          </span>

          <strong>
            {{ formatChartValue(hoveredPoint.metricValue) }}
          </strong>

          <small>
            {{ formatNumber(hoveredPoint.invoice_count) }} faktur
          </small>
        </div>
      </div>
    </section>

    <section class="status-panel">
      <header class="status-header">
        <div>
          <h2>
            System Status
          </h2>

          <p>
            Core services
          </p>
        </div>
      </header>

      <div class="status-list">
        <article class="status-row">
          <div>
            <strong>PostgreSQL</strong>
            <span>Database Service</span>
          </div>

          <span
            :class="[
              'status-badge',
              errorMessage
                ? 'status-warning'
                : 'status-online',
            ]"
          >
            {{ errorMessage ? 'Check' : 'Online' }}
          </span>
        </article>

        <article class="status-row">
          <div>
            <strong>Backend API</strong>
            <span>Golang Service</span>
          </div>

          <span
            :class="[
              'status-badge',
              errorMessage
                ? 'status-warning'
                : 'status-online',
            ]"
          >
            {{ errorMessage ? 'Check' : 'Online' }}
          </span>
        </article>

        <article class="status-row">
          <div>
            <strong>Authentication</strong>
            <span>JWT & RBAC</span>
          </div>

          <span class="status-badge status-online">
            Active
          </span>
        </article>

        <article class="status-row">
          <div>
            <strong>Intelligence</strong>
            <span>Decision Support</span>
          </div>

          <span class="status-badge status-ready">
            Ready
          </span>
        </article>
      </div>
    </section>
  </div>
</template>

<style scoped>
.dashboard-overview {
  width: 100%;
  min-width: 0;
  padding: 1.45rem 1.65rem 2.5rem;
  color: var(--fas-text);
}

.welcome-section {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 2rem;
}

.welcome-copy {
  min-width: 0;
}

.welcome-kicker {
  margin: 0;
  color: var(--fas-accent);
  font-size: 0.95rem;
  font-weight: 760;
}

.welcome-copy h1 {
  margin: 0.38rem 0 0;
  color: var(--fas-text);
  font-size: clamp(
    2.1rem,
    4vw,
    3rem
  );
  font-weight: 850;
  letter-spacing: -0.045em;
}

.role-line {
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  margin-top: 0.55rem;
  color: var(--fas-muted);
  font-size: 0.78rem;
  font-weight: 720;
}

.role-line > span {
  width: 0.48rem;
  height: 0.48rem;
  border-radius: 999px;
  background: var(--fas-accent-hover);
}

.welcome-description {
  max-width: 52rem;
  margin: 1.4rem 0 0;
  color: var(--fas-muted);
  font-size: 0.94rem;
  line-height: 1.7rem;
}

.quick-action-button {
  display: inline-flex;
  min-height: 3.65rem;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 0.9rem;
  border: 1px solid var(--fas-accent-border);
  border-radius: 1rem;
  background: var(--fas-accent);
  padding: 0 1.35rem;
  color: #ffffff;
  font-size: 0.92rem;
  font-weight: 760;
  box-shadow: 0 16px 34px color-mix(
    in srgb,
    var(--fas-accent) 24%,
    transparent
  );
  transition:
    transform 160ms ease,
    background 160ms ease;
}

.quick-action-button:hover {
  background: var(--fas-accent-hover);
  transform: translateY(-1px);
}

.overview-message {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 1.2rem;
  border-radius: 0.9rem;
  padding: 0.85rem 1rem;
  font-size: 0.72rem;
}

.overview-error {
  border: 1px solid rgba(248, 113, 113, 0.24);
  background: rgba(248, 113, 113, 0.08);
  color: #ef4444;
}

.overview-message button {
  min-height: 2rem;
  border: 1px solid currentColor;
  border-radius: 0.55rem;
  padding: 0 0.7rem;
  color: inherit;
  font-size: 0.62rem;
  font-weight: 800;
}

.summary-grid {
  display: grid;
  grid-template-columns:
    repeat(2, minmax(0, 1fr));
  gap: 1.15rem;
  margin-top: 2rem;
}

.summary-card {
  min-width: 0;
  min-height: 12rem;
  border: 1px solid var(--fas-border);
  border-radius: 1.45rem;
  background: var(--fas-card);
  padding: 1.4rem 1.5rem 1.35rem;
  box-shadow: 0 20px 48px var(--fas-shadow);
}

.summary-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.summary-card-head p {
  margin: 0;
  color: var(--fas-muted);
  font-size: 0.9rem;
  font-weight: 650;
}

.summary-icon {
  display: grid;
  width: 3.15rem;
  height: 3.15rem;
  flex: 0 0 auto;
  place-items: center;
  border: 1px solid var(--fas-accent-border);
  border-radius: 1rem;
  background: var(--fas-accent-soft);
  color: var(--fas-accent);
}

.summary-icon svg {
  width: 1.45rem;
  height: 1.45rem;
}

.summary-value {
  display: block;
  margin-top: 0.6rem;
  color: var(--fas-text);
  font-size: 2rem;
  font-weight: 850;
  letter-spacing: -0.035em;
}

.summary-value-money {
  font-size: clamp(
    1.4rem,
    2.3vw,
    2rem
  );
}

.summary-helper {
  display: block;
  margin-top: 1.9rem;
  color: var(--fas-muted-soft);
  font-size: 0.72rem;
}

.performance-panel,
.status-panel {
  margin-top: 1.8rem;
  overflow: hidden;
  border: 1px solid var(--fas-border);
  border-radius: 1.45rem;
  background: var(--fas-card);
  box-shadow: 0 20px 50px var(--fas-shadow);
}

.performance-header,
.status-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1.5rem;
  padding: 1.45rem 1.5rem 1rem;
}

.performance-kicker,
.status-header h2 {
  margin: 0;
  color: var(--fas-text);
  font-size: 1rem;
  font-weight: 820;
}

.performance-header h2,
.status-header p {
  margin: 0.28rem 0 0;
  color: var(--fas-muted);
  font-size: 0.74rem;
  font-weight: 500;
}

.performance-header-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 0.6rem;
}

.period-chip,
.finance-link-button {
  display: inline-flex;
  min-height: 2.8rem;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--fas-border);
  border-radius: 0.85rem;
  background: var(--fas-overlay);
  padding: 0 0.9rem;
  color: var(--fas-muted);
  font-size: 0.72rem;
  font-weight: 700;
}

.finance-link-button {
  gap: 0.45rem;
  border-color: var(--fas-accent-border);
  color: var(--fas-accent-text);
}

.finance-link-button:hover {
  background: var(--fas-accent-soft);
}

.chart-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top: 1px solid var(--fas-border);
  border-bottom: 1px solid var(--fas-border);
  padding: 0.75rem 1.5rem;
}

.metric-tabs,
.chart-live-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 0.45rem;
}

.metric-tab {
  min-height: 2.2rem;
  border: 1px solid var(--fas-border);
  border-radius: 999px;
  background: var(--fas-overlay);
  padding: 0 0.78rem;
  color: var(--fas-muted);
  font-size: 0.62rem;
  font-weight: 780;
  transition:
    border-color 150ms ease,
    background 150ms ease,
    color 150ms ease;
}

.metric-tab:hover,
.metric-tab-active {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.chart-live-meta > span,
.refresh-chart-button {
  display: inline-flex;
  min-height: 2rem;
  align-items: center;
  border: 1px solid var(--fas-border);
  border-radius: 999px;
  background: var(--fas-overlay);
  padding: 0 0.62rem;
  color: var(--fas-muted);
  font-size: 0.56rem;
  font-weight: 760;
}

.chart-live-meta .live-pill {
  border-color: rgba(16, 185, 129, 0.24);
  color: #10b981;
}

.live-pill i {
  width: 0.4rem;
  height: 0.4rem;
  margin-right: 0.3rem;
  border-radius: 999px;
  background: #10b981;
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.1);
}

.refresh-chart-button {
  color: var(--fas-accent-text);
}

.refresh-chart-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.chart-summary-row {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1.5rem 0;
}

.chart-summary-row > div {
  min-width: 0;
}

.chart-summary-label {
  display: block;
  color: var(--fas-muted);
  font-size: 0.62rem;
}

.chart-summary-row strong {
  display: block;
  margin-top: 0.28rem;
  color: var(--fas-text);
  font-size: 1.25rem;
  font-weight: 850;
  letter-spacing: -0.025em;
}

.change-badge {
  display: inline-flex;
  min-height: 1.8rem;
  align-items: center;
  gap: 0.3rem;
  border-radius: 999px;
  padding: 0 0.58rem;
  font-size: 0.56rem;
  font-weight: 850;
}

.change-up {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981;
}

.change-down {
  background: rgba(239, 68, 68, 0.1);
  color: #ef4444;
}

.change-flat {
  background: var(--fas-overlay);
  color: var(--fas-muted);
}

.chart-shell {
  position: relative;
  min-height: 24rem;
  padding: 0.65rem 0.85rem 0.5rem;
}

.overview-chart {
  width: 100%;
  height: 22rem;
  overflow: visible;
  touch-action: none;
}

.chart-grid-lines line {
  stroke: var(--fas-border-strong);
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}

.chart-grid-lines text,
.chart-x-labels text {
  fill: var(--fas-muted);
  font-size: 11px;
  font-family:
    Inter,
    system-ui,
    sans-serif;
}

.chart-volume-bars rect {
  fill: var(--fas-accent-soft);
}

.chart-area-path {
  fill: url(#overview-area-gradient);
}

.chart-line-path {
  fill: none;
  stroke: var(--fas-accent-text);
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
  vector-effect: non-scaling-stroke;
}

.chart-point-layer circle {
  fill: var(--fas-card);
  stroke: var(--fas-accent-text);
  stroke-width: 1.9;
  vector-effect: non-scaling-stroke;
}

.chart-point-layer circle.hovered {
  fill: var(--fas-accent-text);
  stroke: var(--fas-text);
  stroke-width: 2.3;
}

.chart-crosshair line {
  stroke: color-mix(
    in srgb,
    var(--fas-muted) 45%,
    transparent
  );
  stroke-width: 1;
  stroke-dasharray: 4 4;
  vector-effect: non-scaling-stroke;
}

.chart-crosshair circle {
  fill: var(--fas-accent-text);
  stroke: var(--fas-text);
  stroke-width: 2.2;
  vector-effect: non-scaling-stroke;
}

.chart-tooltip {
  position: absolute;
  top: 1.1rem;
  z-index: 5;
  display: flex;
  min-width: 9rem;
  flex-direction: column;
  gap: 0.2rem;
  border: 1px solid var(--fas-accent-border);
  border-radius: 0.8rem;
  background: var(--fas-elevated);
  padding: 0.65rem 0.75rem;
  box-shadow: 0 14px 32px var(--fas-shadow);
  pointer-events: none;
  transform: translateX(-50%);
}

.chart-tooltip span,
.chart-tooltip small {
  color: var(--fas-muted);
  font-size: 0.54rem;
}

.chart-tooltip strong {
  color: var(--fas-accent-text);
  font-size: 0.72rem;
}

.chart-state {
  display: flex;
  min-height: 20rem;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  margin: 0.85rem 1.5rem 1.4rem;
  border: 1px dashed var(--fas-border-strong);
  border-radius: 1rem;
  background: var(--fas-overlay);
  color: var(--fas-muted);
  text-align: center;
}

.chart-state strong {
  color: var(--fas-text-soft);
  font-size: 0.78rem;
}

.chart-state p {
  max-width: 30rem;
  margin: 0;
  color: var(--fas-muted);
  font-size: 0.64rem;
  line-height: 1.1rem;
}

.chart-empty-icon {
  display: grid;
  width: 3rem;
  height: 3rem;
  place-items: center;
  border-radius: 1rem;
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.chart-empty-icon svg {
  width: 1.45rem;
  height: 1.45rem;
}

.overview-spinner {
  width: 1.5rem;
  height: 1.5rem;
  border: 2px solid var(--fas-border-strong);
  border-top-color: var(--fas-accent);
  border-radius: 999px;
  animation: overview-spin 0.8s linear infinite;
}

.status-list {
  display: grid;
  gap: 0.8rem;
  padding: 0.85rem 1.5rem 1.5rem;
}

.status-row {
  display: flex;
  min-height: 5rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid var(--fas-border);
  border-radius: 1rem;
  background: var(--fas-overlay);
  padding: 0.9rem 1rem;
}

.status-row > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.22rem;
}

.status-row strong {
  color: var(--fas-text-soft);
  font-size: 0.8rem;
  font-weight: 760;
}

.status-row span {
  color: var(--fas-muted);
  font-size: 0.66rem;
}

.status-badge {
  display: inline-flex;
  min-width: 4.2rem;
  min-height: 1.9rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  padding: 0 0.62rem;
  font-size: 0.56rem !important;
  font-weight: 820;
}

.status-online {
  background: rgba(16, 185, 129, 0.1);
  color: #10b981 !important;
}

.status-ready {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text) !important;
}

.status-warning {
  background: rgba(245, 158, 11, 0.12);
  color: #f59e0b !important;
}

@keyframes overview-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 980px) {
  .welcome-section {
    align-items: stretch;
    flex-direction: column;
  }

  .quick-action-button {
    align-self: flex-start;
  }

  .summary-grid {
    grid-template-columns: 1fr;
  }

  .chart-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .chart-live-meta {
    justify-content: flex-start;
  }
}

@media (max-width: 720px) {
  .dashboard-overview {
    padding: 1rem;
  }

  .welcome-copy h1 {
    font-size: 2rem;
  }

  .welcome-description {
    margin-top: 1rem;
    font-size: 0.8rem;
    line-height: 1.4rem;
  }

  .performance-header,
  .status-header {
    align-items: stretch;
    flex-direction: column;
  }

  .performance-header-actions {
    justify-content: flex-start;
  }

  .chart-summary-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .overview-chart {
    height: 18rem;
  }

  .chart-shell {
    min-height: 20rem;
    overflow-x: auto;
  }

  .status-row {
    min-height: 4.4rem;
  }
}

@media (max-width: 480px) {
  .summary-card {
    min-height: 10rem;
  }

  .summary-value {
    font-size: 1.6rem;
  }

  .performance-header,
  .chart-toolbar,
  .chart-summary-row,
  .status-header,
  .status-list {
    padding-right: 1rem;
    padding-left: 1rem;
  }

  .period-chip,
  .finance-link-button {
    width: 100%;
  }
}
</style>
