<script setup>
import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import { companyDocumentHTML, companyDocumentRows } from '@/services/company'
import api from '@/services/api'
import FasSelect from '@/components/ui/FasSelect.vue'
import FasIcon from '@/components/ui/FasIcon.vue'

import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from 'vue'

import {
  getFinanceSalesSummary,
  getFinanceSalesTrend,
} from '@/services/finance'

const loading =
  ref(false)

const exporting =
  ref(false)

const errorMessage =
  ref('')

const productPeriod = ref({ date_from: '', date_to: '' })

const lastUpdatedAt =
  ref(null)

const summary =
  ref({})

const trendRows =
  ref([])

const granularity =
  ref('day')

const activePreset =
  ref('THIS_MONTH')

const selectedMetric =
  ref('grand_total')

const hoveredIndex =
  ref(null)

const filters =
  reactive({
    date_from: '',
    date_to: '',
  })

let refreshTimer =
  null

const metricOptions = [
  {
    key: 'grand_total',
    label: 'Total Penjualan',
    shortLabel: 'Penjualan',
    kind: 'currency',
  },
  {
    key: 'net_sales',
    label: 'Penjualan Bersih',
    shortLabel: 'Bersih',
    kind: 'currency',
  },
  {
    key: 'tax_base_amount',
    label: 'DPP',
    shortLabel: 'DPP',
    kind: 'currency',
  },
  {
    key: 'tax_amount',
    label: 'PPN',
    shortLabel: 'PPN',
    kind: 'currency',
  },
  {
    key: 'invoice_count',
    label: 'Jumlah Faktur',
    shortLabel: 'Faktur',
    kind: 'number',
  },
]

const chartWidth =
  1000

const chartHeight =
  360

const chartPadding = {
  left: 76,
  right: 22,
  top: 24,
  bottom: 54,
}

const chartBottom =
  chartHeight -
  chartPadding.bottom

const chartTop =
  chartPadding.top

const chartPlotHeight =
  chartBottom -
  chartTop

const chartPlotWidth =
  chartWidth -
  chartPadding.left -
  chartPadding.right

const toISODate = value => {
  const date =
    new Date(value)

  const year =
    date.getFullYear()

  const month =
    String(
      date.getMonth() + 1,
    ).padStart(
      2,
      '0',
    )

  const day =
    String(
      date.getDate(),
    ).padStart(
      2,
      '0',
    )

  return `${year}-${month}-${day}`
}

const todayISO = () => {
  return toISODate(
    new Date(),
  )
}

const firstDayOfMonthISO = () => {
  const now =
    new Date()

  return toISODate(
    new Date(
      now.getFullYear(),
      now.getMonth(),
      1,
    ),
  )
}

const firstDayOfYearISO = () => {
  const now =
    new Date()

  return toISODate(
    new Date(
      now.getFullYear(),
      0,
      1,
    ),
  )
}

const daysAgoISO = days => {
  const now =
    new Date()

  now.setDate(
    now.getDate() -
      Number(days || 0),
  )

  return toISODate(
    now,
  )
}

const monthsAgoISO = months => {
  const now =
    new Date()

  return toISODate(
    new Date(
      now.getFullYear(),
      now.getMonth() -
        Math.max(
          Number(months || 1) -
            1,
          0,
        ),
      1,
    ),
  )
}

filters.date_from =
  firstDayOfMonthISO()

filters.date_to =
  todayISO()

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
  const amount =
    Number(value || 0)

  if (
    Math.abs(amount) >=
    1_000_000_000
  ) {
    return (
      `Rp ${(
        amount /
        1_000_000_000
      ).toLocaleString(
        'id-ID',
        {
          maximumFractionDigits: 1,
        },
      )} M`
    )
  }

  if (
    Math.abs(amount) >=
    1_000_000
  ) {
    return (
      `Rp ${(
        amount /
        1_000_000
      ).toLocaleString(
        'id-ID',
        {
          maximumFractionDigits: 1,
        },
      )} jt`
    )
  }

  if (
    Math.abs(amount) >=
    1_000
  ) {
    return (
      `Rp ${(
        amount /
        1_000
      ).toLocaleString(
        'id-ID',
        {
          maximumFractionDigits: 1,
        },
      )} rb`
    )
  }

  return formatCurrency(
    amount,
  )
}

const formatNumber = value => {
  return new Intl.NumberFormat(
    'id-ID',
  ).format(
    Number(value || 0),
  )
}

const formatDate = value => {
  if (!value) {
    return '—'
  }

  const date =
    new Date(value)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return String(value)
  }

  return new Intl.DateTimeFormat(
    'id-ID',
    {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
    },
  ).format(date)
}

const formatDateTime = value => {
  if (!value) {
    return '—'
  }

  const date =
    new Date(value)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return String(value)
  }

  return new Intl.DateTimeFormat(
    'id-ID',
    {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    },
  ).format(date)
}

const formatPeriod = value => {
  const date =
    new Date(value)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return String(value || '—')
  }

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

const currentMetric =
  computed(() => {
    return (
      metricOptions.find(
        item =>
          item.key ===
          selectedMetric.value,
      ) ||
      metricOptions[0]
    )
  })

const formatMetricValue = value => {
  if (
    currentMetric.value.kind ===
    'number'
  ) {
    return formatNumber(
      value,
    )
  }

  return formatCurrency(
    value,
  )
}

const formatMetricCompact = value => {
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

const chartData =
  computed(() => {
    return trendRows.value.map(
      row => ({
        ...row,
        metricValue:
          Number(
            row[
              selectedMetric.value
            ] ||
            0,
          ),
      }),
    )
  })

const chartMax =
  computed(() => {
    const values =
      chartData.value.map(
        item =>
          item.metricValue,
      )

    const max =
      Math.max(
        0,
        ...values,
      )

    if (max <= 0) {
      return 1
    }

    return max * 1.08
  })

const chartMaxInvoices =
  computed(() => {
    const max =
      Math.max(
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

const chartPoints =
  computed(() => {
    return chartData.value.map(
      (
        item,
        index,
      ) => ({
        ...item,
        x:
          pointX(index),
        y:
          pointY(
            item.metricValue,
          ),
      }),
    )
  })

const chartLinePath =
  computed(() => {
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

const chartAreaPath =
  computed(() => {
    const points =
      chartPoints.value

    if (
      points.length ===
      0
    ) {
      return ''
    }

    const first =
      points[0]

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

const yTicks =
  computed(() => {
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

const xLabels =
  computed(() => {
    const data =
      chartData.value

    if (
      data.length ===
      0
    ) {
      return []
    }

    const desired =
      Math.min(
        6,
        data.length,
      )

    if (
      desired === 1
    ) {
      return [
        {
          index: 0,
          x:
            pointX(0),
          label:
            formatPeriod(
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
        x:
          pointX(index),
        label:
          formatPeriod(
            data[index]
              .period_start,
          ),
      }),
    )
  })

const hoveredPoint =
  computed(() => {
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

const chartChange =
  computed(() => {
    if (
      chartData.value.length ===
      0
    ) {
      return {
        amount: 0,
        percent: 0,
        direction: 'flat',
        hasPercent: true,
      }
    }

    const first =
      Number(
        chartData.value[0]
          .metricValue ||
        0,
      )

    const last =
      Number(
        chartData.value[
          chartData.value.length -
            1
        ].metricValue ||
        0,
      )

    const amount =
      last - first

    let percent =
      0

    let hasPercent =
      true

    if (
      first === 0 &&
      last !== 0
    ) {
      hasPercent =
        false
    } else if (
      first !== 0
    ) {
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

const latestMetricValue =
  computed(() => {
    if (
      chartData.value.length ===
      0
    ) {
      return 0
    }

    return Number(
      chartData.value[
        chartData.value.length -
          1
      ].metricValue ||
      0,
    )
  })

const highestPoint =
  computed(() => {
    if (
      chartData.value.length ===
      0
    ) {
      return null
    }

    return chartData.value.reduce(
      (
        best,
        item,
      ) =>
        item.metricValue >
        best.metricValue
          ? item
          : best,
      chartData.value[0],
    )
  })

const averageMetric =
  computed(() => {
    if (
      chartData.value.length ===
      0
    ) {
      return 0
    }

    return (
      chartData.value.reduce(
        (
          total,
          item,
        ) =>
          total +
          item.metricValue,
        0,
      ) /
      chartData.value.length
    )
  })

const periodLabel =
  computed(() => {
    return (
      `${formatDate(
        filters.date_from,
      )} — ` +
      `${formatDate(
        filters.date_to,
      )}`
    )
  })

const granularityLabel =
  computed(() => {
    if (
      granularity.value ===
      'month'
    ) {
      return 'Bulanan'
    }

    if (
      granularity.value ===
      'week'
    ) {
      return 'Mingguan'
    }

    return 'Harian'
  })

const termPercent =
  computed(() => {
    const total =
      Number(
        summary.value
          .invoice_count ||
        0,
      )

    if (total <= 0) {
      return 0
    }

    return Math.round(
      (
        Number(
          summary.value
            .term_count ||
          0,
        ) /
        total
      ) *
        100,
    )
  })

const immediatePercent =
  computed(() => {
    const total =
      Number(
        summary.value
          .invoice_count ||
        0,
      )

    if (total <= 0) {
      return 0
    }

    return Math.round(
      (
        Number(
          summary.value
            .immediate_count ||
          0,
        ) /
        total
      ) *
        100,
    )
  })

const printedPercent =
  computed(() => {
    const total =
      Number(
        summary.value
          .invoice_count ||
        0,
      )

    if (total <= 0) {
      return 0
    }

    return Math.round(
      (
        Number(
          summary.value
            .printed_count ||
          0,
        ) /
        total
      ) *
        100,
    )
  })

const requestParams =
  computed(() => ({
    date_from:
      filters.date_from,
    date_to:
      filters.date_to,
    print_state:
      'all',
  }))

let analyticsRequestID = 0
let analyticsDisposed = false

const loadAnalytics = async ({ silent = false } = {}) => {
  const params = { ...requestParams.value }
  if (!params.date_from || !params.date_to || params.date_from > params.date_to) {
    errorMessage.value = 'Isi tanggal awal dan akhir dengan urutan yang benar.'
    return
  }
  const id = ++analyticsRequestID
  if (!silent) loading.value = true
  errorMessage.value = ''
  const samePeriod = productPeriod.value.date_from === params.date_from && productPeriod.value.date_to === params.date_to
  productPeriod.value = params
  if (samePeriod) productTrendLoad()

  try {
    const [summaryResponse, trendResponse] = await Promise.all([
      getFinanceSalesSummary(params),
      getFinanceSalesTrend(params),
    ])
    if (analyticsDisposed || id !== analyticsRequestID) return
    summary.value = summaryResponse?.data || {}
    trendRows.value = Array.isArray(trendResponse?.data) ? trendResponse.data : []
    granularity.value = trendResponse?.granularity || 'day'
    lastUpdatedAt.value = new Date()
    if (hoveredIndex.value !== null && hoveredIndex.value >= trendRows.value.length) hoveredIndex.value = null
  } catch (error) {
    if (analyticsDisposed || id !== analyticsRequestID) return
    errorMessage.value = error.response?.data?.error || error.message || 'Analytics Finance belum dapat dimuat.'
  } finally {
    if (!analyticsDisposed && id === analyticsRequestID) loading.value = false
  }
}

onBeforeUnmount(() => { analyticsDisposed = true; analyticsRequestID++ })

const applyPreset = async preset => {
  activePreset.value =
    preset

  filters.date_to =
    todayISO()

  if (
    preset ===
    'TODAY'
  ) {
    filters.date_from =
      todayISO()
  }

  if (
    preset ===
    'THIS_MONTH'
  ) {
    filters.date_from =
      firstDayOfMonthISO()
  }

  if (
    preset ===
    'ONE_MONTH'
  ) {
    filters.date_from =
      daysAgoISO(
        29,
      )
  }

  if (
    preset ===
    'THREE_MONTHS'
  ) {
    filters.date_from =
      monthsAgoISO(
        3,
      )
  }

  if (
    preset ===
    'SIX_MONTHS'
  ) {
    filters.date_from =
      monthsAgoISO(
        6,
      )
  }

  if (
    preset ===
    'TWELVE_MONTHS'
  ) {
    filters.date_from =
      monthsAgoISO(
        12,
      )
  }

  if (
    preset ===
    'THIS_YEAR'
  ) {
    filters.date_from =
      firstDayOfYearISO()
  }

  hoveredIndex.value =
    null

  await loadAnalytics()
}

const applyCustomPeriod = async () => {
  activePreset.value =
    'CUSTOM'

  hoveredIndex.value =
    null

  await loadAnalytics()
}

const selectMetric = key => {
  selectedMetric.value =
    key

  hoveredIndex.value =
    null
}

const handleChartPointerMove = event => {
  const total =
    chartPoints.value.length

  if (total <= 0) {
    hoveredIndex.value =
      null
    return
  }

  const rect =
    event.currentTarget
      .getBoundingClientRect()

  if (
    rect.width <= 0
  ) {
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
    hoveredIndex.value =
      0
    return
  }

  const normalized =
    (
      viewX -
      chartPadding.left
    ) /
    chartPlotWidth

  const index =
    Math.round(
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
  hoveredIndex.value =
    null
}

const exportTrendExcel = async () => {
  exporting.value =
    true

  errorMessage.value = ''

  try {
    const rows =
      trendRows.value

    const body =
      rows
        .map(
          (
            row,
            index,
          ) => `
            <tr>
              <td>${index + 1}</td>
              <td>${formatDate(
                row.period_start,
              )}</td>
              <td>${Number(
                row.invoice_count ||
                0,
              )}</td>
              <td>${Number(
                row.customer_count ||
                0,
              )}</td>
              <td>${Number(
                row.total_qty ||
                0,
              )}</td>
              <td>${Number(
                row.subtotal ||
                0,
              )}</td>
              <td>${Number(
                row.discount_amount ||
                0,
              )}</td>
              <td>${Number(
                row.net_sales ||
                0,
              )}</td>
              <td>${Number(
                row.tax_base_amount ||
                0,
              )}</td>
              <td>${Number(
                row.tax_amount ||
                0,
              )}</td>
              <td>${Number(
                row.grand_total ||
                0,
              )}</td>
            </tr>
          `,
        )
        .join('')

    const workbook = `
      <html
        xmlns:o="urn:schemas-microsoft-com:office:office"
        xmlns:x="urn:schemas-microsoft-com:office:excel"
        xmlns="http://www.w3.org/TR/REC-html40"
      >
      <head>
        <meta charset="UTF-8" />
      </head>

      <body>${companyDocumentHTML()}
        <table border="1">
          <tr>
            <th colspan="11">
              PT FURI ANUGERAH SEJAHTERA - ANALYTICS FINANCE
            </th>
          </tr>

          <tr>
            <td colspan="11">
              Periode ${filters.date_from} s/d ${filters.date_to}
            </td>
          </tr>

          <tr>
            <th>No</th>
            <th>Periode</th>
            <th>Faktur</th>
            <th>Customer</th>
            <th>Qty</th>
            <th>Subtotal</th>
            <th>Diskon</th>
            <th>Penjualan Bersih</th>
            <th>DPP</th>
            <th>PPN</th>
            <th>Total Penjualan</th>
          </tr>

          ${body}
        </table>
      </body>
      </html>
    `

    const blob =
      new Blob(
        [
          '\ufeff',
          workbook,
        ],
        {
          type:
            'application/vnd.ms-excel;charset=utf-8;',
        },
      )

    const url =
      URL.createObjectURL(
        blob,
      )

    const anchor =
      document.createElement(
        'a',
      )

    anchor.href =
      url

    anchor.download =
      `analytics-finance-${filters.date_from}-${filters.date_to}.xls`

    document.body.appendChild(
      anchor,
    )

    anchor.click()
    anchor.remove()

    URL.revokeObjectURL(
      url,
    )
  } catch (error) {
    errorMessage.value =
      error.message ||
      'Analytics belum dapat diekspor.'
  } finally {
    exporting.value =
      false
  }
}

const printAnalytics = () => {
  if (loading.value || exporting.value) {
    return
  }

  errorMessage.value = ''

  const win =
    window.open(
      '',
      '_blank',
    )

  if (!win) {
    errorMessage.value =
      'Izinkan pop-up browser untuk mencetak Analytics.'
    return
  }

  try {
    win.opener = null

    const rows =
      trendRows.value
        .map(
          (
            row,
            index,
          ) => `
            <tr>
              <td>${index + 1}</td>
              <td>${formatDate(
                row.period_start,
              )}</td>
              <td>${formatNumber(
                row.invoice_count,
              )}</td>
              <td>${formatNumber(
                row.customer_count,
              )}</td>
              <td>${formatNumber(
                row.total_qty,
              )}</td>
              <td>${formatCurrency(
                row.net_sales,
              )}</td>
              <td>${formatCurrency(
                row.tax_base_amount,
              )}</td>
              <td>${formatCurrency(
                row.tax_amount,
              )}</td>
              <td>${formatCurrency(
                row.grand_total,
              )}</td>
            </tr>
          `,
        )
        .join('')

    win.document.open()

    win.document.write(`
      <!DOCTYPE html>
      <html lang="id">
      <head>
        <meta charset="UTF-8" />
        <title>Analytics Finance PT FAS</title>

        <style>
          @page {
            size: A4 landscape;
            margin: 10mm;
          }

          * {
            box-sizing: border-box;
          }

          body {
            margin: 0;
            color: #111827;
            font-family:
              Arial,
              Helvetica,
              sans-serif;
            font-size: 9px;
          }

          .header {
            display: flex;
            justify-content: space-between;
            gap: 24px;
            border-bottom: 2px solid #111827;
            padding-bottom: 8px;
          }

          h1 {
            margin: 0;
            font-size: 18px;
          }

          h2 {
            margin: 4px 0 0;
            font-size: 12px;
          }

          .meta {
            text-align: right;
            line-height: 1.55;
          }

          .summary {
            display: grid;
            grid-template-columns:
              repeat(4, 1fr);
            gap: 7px;
            margin: 12px 0;
          }

          .card {
            border: 1px solid #cbd5e1;
            border-radius: 5px;
            padding: 7px;
          }

          .card span {
            display: block;
            color: #64748b;
          }

          .card strong {
            display: block;
            margin-top: 4px;
            font-size: 12px;
          }

          table {
            width: 100%;
            border-collapse: collapse;
          }

          th,
          td {
            border: 1px solid #94a3b8;
            padding: 4px 5px;
            text-align: left;
          }

          th {
            background: #f1f5f9;
          }
        .product-trend .table-scroll{max-height:min(65vh,42rem)}.product-trend thead{position:sticky;top:0;z-index:1}.product-trend footer div{align-items:center;flex-wrap:wrap}.product-trend .page-size{display:flex;flex-direction:column;gap:.3rem;min-width:9rem}.product-trend .page-size>span{font-size:.7rem}
</style>
      </head>

      <body>${companyDocumentHTML()}
        <div class="header">
          <div>
            <h1>
              PT FURI ANUGERAH SEJAHTERA
            </h1>

            <h2>
              ANALYTICS FINANCE
            </h2>
          </div>

          <div class="meta">
            Periode:
            ${filters.date_from}
            s/d
            ${filters.date_to}
            <br />

            Granularitas:
            ${granularityLabel.value}
            <br />

            Dicetak:
            ${formatDateTime(
              new Date(),
            )}
          </div>
        </div>

        <section class="summary">
          <div class="card">
            <span>Total Penjualan</span>
            <strong>
              ${formatCurrency(
                summary.value
                  .grand_total,
              )}
            </strong>
          </div>

          <div class="card">
            <span>Jumlah Faktur</span>
            <strong>
              ${formatNumber(
                summary.value
                  .invoice_count,
              )}
            </strong>
          </div>

          <div class="card">
            <span>DPP</span>
            <strong>
              ${formatCurrency(
                summary.value
                  .tax_base_amount,
              )}
            </strong>
          </div>

          <div class="card">
            <span>PPN</span>
            <strong>
              ${formatCurrency(
                summary.value
                  .tax_amount,
              )}
            </strong>
          </div>
        </section>

        <table>
          <thead>
            <tr>
              <th>No</th>
              <th>Periode</th>
              <th>Faktur</th>
              <th>Customer</th>
              <th>Qty</th>
              <th>Penjualan Bersih</th>
              <th>DPP</th>
              <th>PPN</th>
              <th>Total Penjualan</th>
            </tr>
          </thead>

          <tbody>
            ${rows}
          </tbody>
        </table>

        <script>
          window.onload = () => {
            window.print()
          }
        <\/script>
      </body>
      </html>
    `)

    win.document.close()
  } catch (error) {
    win.close()
    errorMessage.value = error.message || 'Dokumen Analytics belum dapat disiapkan.'
  }
}

onMounted(
  async () => {
    await loadAnalytics()

    refreshTimer =
      window.setInterval(
        () => {
          loadAnalytics({
            silent: true,
          })
        },
        15000,
      )

    await nextTick()
  },
)

onBeforeUnmount(
  () => {
    if (refreshTimer) {
      window.clearInterval(
        refreshTimer,
      )
    }
  },
)

const productTrendFetch = async params => {
  const response = await api.get('/finance/products/trend', { params })
  return response.data
}

const productTrendCategories = [['ALL', 'Semua produk'], ['FAST', 'Laris'], ['MEDIUM', 'Laku sedang'], ['UNSOLD', 'Belum terjual']]
const productTrendSortOptions = [['frequency', 'Paling sering dibeli'], ['quantity', 'Jumlah terjual terbesar'], ['revenue', 'Nilai penjualan terbesar'], ['name', 'Nama produk A–Z']]
const productTrendCategory = ref('ALL')
const productTrendSort = ref('frequency')
const productTrendQuery = ref('')
const productTrendMonth = ref(todayISO().slice(0, 7))
const productTrendRows = ref([])
const productTrendSummary = ref({})
const productTrendTotal = ref(0)
const productTrendOffset = ref(0)
const productTrendLimit = ref(15)
const productTrendLoading = ref(false)
const productTrendExporting = ref(false)
const productTrendError = ref('')
const productTrendReportError = ref('')
const productTrendApplied = ref(null)
let productTrendRequestID = 0
let productTrendSearchTimer
let productTrendDisposed = false
const productTrendPage = computed(() => Math.floor(productTrendOffset.value / productTrendLimit.value) + 1)
const productTrendPages = computed(() => Math.max(1, Math.ceil(productTrendTotal.value / productTrendLimit.value)))
const productTrendCount = value => new Intl.NumberFormat('id-ID').format(Number(value || 0))
const productTrendMoney = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 2 }).format(Number(value || 0))
const productTrendLabel = value => productTrendCategories.find(item => item[0] === value)?.[1] || value
const productTrendDate = value => value ? new Date(value.slice(0, 10) + 'T12:00:00+07:00').toLocaleDateString('id-ID', { day: 'numeric', month: 'short', year: 'numeric', timeZone: 'Asia/Jakarta' }) : 'Belum terjual'
const productTrendPeriodLabel = computed(() => productPeriod.value.date_from && productPeriod.value.date_to ? `${productTrendDate(productPeriod.value.date_from)} – ${productTrendDate(productPeriod.value.date_to)}` : 'Pilih periode')
const productTrendCanReport = computed(() => !productTrendLoading.value && !productTrendExporting.value && !productTrendError.value && productTrendTotal.value > 0)
const productTrendLoad = async (reset = false) => {
  if (!productPeriod.value.date_from || !productPeriod.value.date_to) return
  if (reset) productTrendOffset.value = 0
  const id = ++productTrendRequestID
  const snapshot = { date_from: productPeriod.value.date_from, date_to: productPeriod.value.date_to, query: productTrendQuery.value, category: productTrendCategory.value, sort: productTrendSort.value }
  productTrendLoading.value = true
  productTrendError.value = ''
  try {
    const result = await productTrendFetch({ ...snapshot, limit: Number(productTrendLimit.value), offset: productTrendOffset.value })
    if (productTrendDisposed || id !== productTrendRequestID) return
    productTrendRows.value = result.data || []
    productTrendSummary.value = result.summary || {}
    productTrendTotal.value = Number(result.meta?.total || 0)
    productTrendApplied.value = snapshot
    if (productTrendOffset.value >= productTrendTotal.value && productTrendOffset.value > 0) {
      productTrendOffset.value = 0
      return productTrendLoad()
    }
  } catch (failure) {
    if (productTrendDisposed || id !== productTrendRequestID) return
    productTrendRows.value = []
    productTrendSummary.value = {}
    productTrendTotal.value = 0
    productTrendError.value = failure.response?.data?.error || 'Statistik produk belum tersedia. Pastikan backend Finance sudah diperbarui dan dijalankan ulang.'
  } finally {
    if (!productTrendDisposed && id === productTrendRequestID) productTrendLoading.value = false
  }
}
const productTrendChangePage = direction => {
  if (productTrendLoading.value || productTrendExporting.value) return
  const next = productTrendOffset.value + direction * Number(productTrendLimit.value)
  if (next >= 0 && next < productTrendTotal.value) { productTrendOffset.value = next; productTrendLoad() }
}
const productTrendApplyMonth = async () => {
  if (!/^\d{4}-\d{2}$/.test(productTrendMonth.value)) { productTrendReportError.value = 'Pilih bulan yang ingin dilihat.'; return }
  const [year, month] = productTrendMonth.value.split('-').map(Number)
  if (month < 1 || month > 12 || productTrendMonth.value > todayISO().slice(0, 7)) { productTrendReportError.value = 'Pilih bulan yang sudah berjalan.'; return }
  const end = toISODate(new Date(year, month, 0))
  filters.date_from = `${productTrendMonth.value}-01`
  filters.date_to = end > todayISO() ? todayISO() : end
  productTrendReportError.value = ''
  await applyCustomPeriod()
}
const productTrendCollect = async snapshot => {
  const items = []
  let offset = 0, total = Infinity
  while (offset < total) {
    const result = await productTrendFetch({ ...snapshot, offset, limit: 1000 })
    const batch = result.data || []
    total = Number(result.meta?.total || 0)
    if (!batch.length) break
    items.push(...batch)
    offset += batch.length
  }
  return items
}
const productTrendReportHeadings = ['Peringkat kelompok', 'Nama produk', 'Kode', 'SKU', 'Kelompok', 'Frekuensi faktur', 'Jumlah terjual', 'Unit', 'Stok kini', 'Stok minimum', 'Harga beli per unit (Rp)', 'Nilai penjualan item (Rp)', 'Terakhir terjual']
const productTrendReportRows = rows => rows.map(row => [Number(row.category_rank), row.name, row.product_code, row.sku, productTrendLabel(row.category), Number(row.invoice_count), Number(row.qty), row.unit, Number(row.stock), Number(row.min_stock), Number(row.purchase_price), Number(row.net_sales), row.last_sold || 'Belum terjual'])
const productTrendReportDescription = snapshot => [
  `${productTrendDate(snapshot.date_from)} – ${productTrendDate(snapshot.date_to)}`,
  productTrendLabel(snapshot.category),
  productTrendSortOptions.find(item => item[0] === snapshot.sort)?.[1],
  snapshot.query ? `Pencarian: ${snapshot.query}` : 'Semua nama produk',
  'Stok dan harga beli adalah posisi saat laporan diambil, bukan riwayat stok periode tersebut.',
].join(' · ')

const productTrendWorkbook = rows => {
const xml = value => String(value ?? '').replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/g, '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&apos;' }[c]))
const encoder = new TextEncoder()
const column = index => {
  let name = ''
  for (let n = index + 1; n > 0; n = Math.floor((n - 1) / 26)) name = String.fromCharCode(65 + (n - 1) % 26) + name
  return name
}
const crc32 = data => {
  let crc = 0xffffffff
  for (const value of data) {
    crc ^= value
    for (let bit = 0; bit < 8; bit++) crc = (crc >>> 1) ^ ((crc & 1) ? 0xedb88320 : 0)
  }
  return (crc ^ 0xffffffff) >>> 0
}
const header = size => ({ bytes: new Uint8Array(size), view: null })

// Small uncompressed ZIP writer: Office Open XML, no external package or executable formulas.
const zip = files => {
  const chunks = [], central = []
  let offset = 0
  for (const [path, text] of files) {
    const name = encoder.encode(path), data = encoder.encode(text), crc = crc32(data)
    const local = header(30 + name.length); local.view = new DataView(local.bytes.buffer)
    local.view.setUint32(0, 0x04034b50, true); local.view.setUint16(4, 20, true)
    local.view.setUint16(6, 0x800, true); local.view.setUint16(12, 0x21, true)
    local.view.setUint32(14, crc, true); local.view.setUint32(18, data.length, true)
    local.view.setUint32(22, data.length, true); local.view.setUint16(26, name.length, true)
    local.bytes.set(name, 30)
    chunks.push(local.bytes, data)
    const directory = header(46 + name.length); directory.view = new DataView(directory.bytes.buffer)
    directory.view.setUint32(0, 0x02014b50, true); directory.view.setUint16(4, 20, true); directory.view.setUint16(6, 20, true)
    directory.view.setUint16(8, 0x800, true); directory.view.setUint16(14, 0x21, true)
    directory.view.setUint32(16, crc, true); directory.view.setUint32(20, data.length, true); directory.view.setUint32(24, data.length, true)
    directory.view.setUint16(28, name.length, true); directory.view.setUint32(42, offset, true)
    directory.bytes.set(name, 46); central.push(directory.bytes)
    offset += local.bytes.length + data.length
  }
  const size = central.reduce((n, part) => n + part.length, 0)
  const end = header(22); end.view = new DataView(end.bytes.buffer)
  end.view.setUint32(0, 0x06054b50, true); end.view.setUint16(8, files.length, true); end.view.setUint16(10, files.length, true)
  end.view.setUint32(12, size, true); end.view.setUint32(16, offset, true)
  return new Blob([...chunks, ...central, end.bytes], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
}

const buildFinanceXlsx = rows => {
  const cells = rows.map((row, r) => `<row r="${r + 1}">${row.map((value, c) => typeof value === 'number' && Number.isFinite(value) ? `<c r="${column(c)}${r + 1}"><v>${value}</v></c>` : `<c r="${column(c)}${r + 1}" t="inlineStr"><is><t xml:space="preserve">${xml(value)}</t></is></c>`).join('')}</row>`).join('')
  return zip([
    ['[Content_Types].xml', '<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>'],
    ['_rels/.rels', '<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>'],
    ['xl/workbook.xml', '<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Tren Produk" sheetId="1" r:id="rId1"/></sheets></workbook>'],
    ['xl/_rels/workbook.xml.rels', '<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>'],
    ['xl/worksheets/sheet1.xml', `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${cells}</sheetData></worksheet>`],
  ])
}


return buildFinanceXlsx(rows)
}
const productTrendExport = async kind => {
  if (!productTrendCanReport.value || !productTrendApplied.value) return
  let popup
  const snapshot = { ...productTrendApplied.value }
  productTrendExporting.value = true
  productTrendReportError.value = ''
  try {
    if (kind === 'print') {
      popup = window.open('', '_blank')
      if (!popup) throw new Error('Izinkan pop-up browser untuk membuka dokumen cetak.')
      popup.opener = null
      popup.document.write('<p>Menyiapkan laporan tren produk…</p>')
    }
    const items = await productTrendCollect(snapshot)
    const description = productTrendReportDescription(snapshot)
    const rows = productTrendReportRows(items)
    if (kind === 'print') {
      const escape = value => String(value ?? '').replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]))
      popup.document.open()
      popup.onload = () => { popup.focus(); popup.print() }
      popup.document.write(`<!doctype html><html lang="id"><head><meta charset="UTF-8"><title>PT FAS — Tren Produk</title><style>@page{size:A4 landscape;margin:10mm}body{font:10px Arial,sans-serif;color:#172033}h1{font-size:19px}p{line-height:1.6}table{width:100%;border-collapse:collapse}th,td{border:1px solid #cbd5e1;padding:5px;text-align:left}th{background:#f1f5f9}thead{display:table-header-group}tr{break-inside:avoid}@media print{button{display:none}}.product-trend .table-scroll{max-height:min(65vh,42rem)}.product-trend thead{position:sticky;top:0;z-index:1}.product-trend footer div{align-items:center;flex-wrap:wrap}.product-trend .page-size{display:flex;flex-direction:column;gap:.3rem;min-width:9rem}.product-trend .page-size>span{font-size:.7rem}
</style></head><body>${companyDocumentHTML()}<button onclick="window.print()">Cetak / Simpan PDF</button><h1>PT FAS — Tren Produk untuk Pembelian</h1><p>${escape(description)}</p><table><thead><tr>${productTrendReportHeadings.map(value => `<th>${escape(value)}</th>`).join('')}</tr></thead><tbody>${rows.map(row => `<tr>${row.map(cell => `<td>${escape(cell)}</td>`).join('')}</tr>`).join('')}</tbody></table></body></html>`)
      popup.document.close()
    } else {
      const blob = productTrendWorkbook([...companyDocumentRows(), [], ['PT FAS — Tren Produk'], [description], productTrendReportHeadings, ...rows])
      const url = URL.createObjectURL(blob), link = document.createElement('a')
      link.href = url
      link.download = `PTFAS_Tren_Produk_${snapshot.date_from}_${snapshot.date_to}.xlsx`
      document.body.appendChild(link); link.click(); link.remove()
      setTimeout(() => URL.revokeObjectURL(url), 1000)
    }
  } catch (failure) {
    popup?.close()
    productTrendReportError.value = failure.response?.data?.error || failure.message || 'Laporan tren produk belum dapat dibuat.'
  } finally { productTrendExporting.value = false }
}
watch(() => [productPeriod.value.date_from, productPeriod.value.date_to], () => productTrendLoad(true))
watch(() => [productTrendCategory.value, productTrendSort.value, productTrendLimit.value], () => productTrendLoad(true))
watch(productTrendQuery, () => { clearTimeout(productTrendSearchTimer); productTrendSearchTimer = setTimeout(() => productTrendLoad(true), 350) })
onBeforeUnmount(() => { productTrendDisposed = true; productTrendRequestID++; clearTimeout(productTrendSearchTimer) })

</script>

<template>
  <div class="analytics-page">
    <header class="analytics-header">
      <div>
        <div class="live-status">
          <span class="live-dot"></span>

          <span>
            LIVE FINANCE DATA
          </span>

          <span class="live-divider">
            ·
          </span>

          <span>
            Refresh otomatis 15 detik
          </span>
        </div>

        <h1>
          Finance Analytics
        </h1>

        <p>
          Pantau pergerakan penjualan PT FAS seperti dashboard market.
          Semua angka berasal dari faktur Distribution yang tersimpan.
        </p>
      </div>

      <div class="header-actions">
        <button
          type="button"
          class="action-button secondary"
          :disabled="
            loading ||
            exporting
          "
          @click="
            loadAnalytics()
          "
        >
          <FasIcon name="refresh" />
          Refresh
        </button>

        <button
          type="button"
          class="action-button secondary"
          :disabled="
            loading ||
            exporting
          "
          @click="
            printAnalytics
          "
        >
          Print / PDF
        </button>

        <button
          type="button"
          class="action-button primary"
          :disabled="
            exporting
          "
          @click="
            exportTrendExcel
          "
        >
          Export Excel
        </button>
      </div>
    </header>

    <FinanceNavigation />

    <div
      v-if="
        errorMessage
      "
      class="message-error"
    >
      {{ errorMessage }}
    </div>

    <section class="period-panel">
      <div class="preset-list">
        <button
          :class="{
            active:
              activePreset ===
              'TODAY',
          }"
          @click="
            applyPreset(
              'TODAY',
            )
          "
        >
          Hari Ini
        </button>

        <button
          :class="{
            active:
              activePreset ===
              'THIS_MONTH',
          }"
          @click="
            applyPreset(
              'THIS_MONTH',
            )
          "
        >
          Bulan Ini
        </button>

        <button
          :class="{
            active:
              activePreset ===
              'ONE_MONTH',
          }"
          @click="
            applyPreset(
              'ONE_MONTH',
            )
          "
        >
          1 Bulan
        </button>

        <button
          :class="{
            active:
              activePreset ===
              'THREE_MONTHS',
          }"
          @click="
            applyPreset(
              'THREE_MONTHS',
            )
          "
        >
          3 Bulan
        </button>

        <button
          :class="{
            active:
              activePreset ===
              'SIX_MONTHS',
          }"
          @click="
            applyPreset(
              'SIX_MONTHS',
            )
          "
        >
          6 Bulan
        </button>

        <button
          :class="{
            active:
              activePreset ===
              'TWELVE_MONTHS',
          }"
          @click="
            applyPreset(
              'TWELVE_MONTHS',
            )
          "
        >
          1 Tahun
        </button>

        <button
          :class="{
            active:
              activePreset ===
              'THIS_YEAR',
          }"
          @click="
            applyPreset(
              'THIS_YEAR',
            )
          "
        >
          Tahun Ini
        </button>
      </div>

      <div class="custom-period">
        <label>
          <span>
            Dari
          </span>

          <input
            v-model="
              filters.date_from
            "
            type="date"
            @change="
              activePreset =
                'CUSTOM'
            "
          />
        </label>

        <span class="arrow">
          →
        </span>

        <label>
          <span>
            Sampai
          </span>

          <input
            v-model="
              filters.date_to
            "
            type="date"
            @change="
              activePreset =
                'CUSTOM'
            "
          />
        </label>

        <button
          type="button"
          class="apply-button"
          @click="
            applyCustomPeriod
          "
        >
          Terapkan
        </button>
      </div>
    </section>

    <section class="summary-grid">
      <article class="summary-card featured">
        <div>
          <span>
            Total Penjualan
          </span>

          <small>
            {{ periodLabel }}
          </small>
        </div>

        <strong>
          {{
            formatCompactCurrency(
              summary.grand_total,
            )
          }}
        </strong>

        <p>
          {{
            formatCurrency(
              summary.grand_total,
            )
          }}
        </p>
      </article>

      <article class="summary-card">
        <div>
          <span>
            Jumlah Faktur
          </span>

          <small>
            transaksi periode aktif
          </small>
        </div>

        <strong>
          {{
            formatNumber(
              summary.invoice_count,
            )
          }}
        </strong>

        <p>
          {{
            formatNumber(
              summary.total_qty,
            )
          }}
          total qty
        </p>
      </article>

      <article class="summary-card">
        <div>
          <span>
            Customer
          </span>

          <small>
            customer unik
          </small>
        </div>

        <strong>
          {{
            formatNumber(
              summary.customer_count,
            )
          }}
        </strong>

        <p>
          rata-rata faktur
          {{
            formatCompactCurrency(
              summary.average_invoice,
            )
          }}
        </p>
      </article>

      <article class="summary-card">
        <div>
          <span>
            PPN
          </span>

          <small>
            pajak periode aktif
          </small>
        </div>

        <strong>
          {{
            formatCompactCurrency(
              summary.tax_amount,
            )
          }}
        </strong>

        <p>
          DPP
          {{
            formatCurrency(
              summary.tax_base_amount,
            )
          }}
        </p>
      </article>
    </section>

    <section class="market-panel">
      <header class="market-header">
        <div>
          <div class="market-kicker">
            REAL-TIME MARKET VIEW
          </div>

          <h2>
            Pergerakan
            {{
              currentMetric.label
            }}
          </h2>

          <div class="market-value-row">
            <strong>
              {{
                formatMetricValue(
                  latestMetricValue,
                )
              }}
            </strong>

            <span
              :class="[
                'change-badge',
                `change-${chartChange.direction}`,
              ]"
            >
              <template
                v-if="
                  chartChange.direction ===
                  'up'
                "
              >
                ▲
              </template>

              <template
                v-else-if="
                  chartChange.direction ===
                  'down'
                "
              >
                ▼
              </template>

              <template
                v-else
              >
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

          <p>
            Perubahan dari titik awal ke titik terakhir pada periode aktif.
          </p>
        </div>

        <div class="market-meta">
          <span class="live-pill">
            <i></i>
            LIVE
          </span>

          <span>
            {{ granularityLabel }}
          </span>

          <span>
            {{ trendRows.length }}
            titik data
          </span>

          <span>
            {{
              lastUpdatedAt
                ? `Update ${formatDateTime(
                    lastUpdatedAt,
                  )}`
                : 'Belum diperbarui'
            }}
          </span>
        </div>
      </header>

      <div class="metric-tabs">
        <button
          v-for="
            metric in metricOptions
          "
          :key="
            metric.key
          "
          :class="{
            active:
              selectedMetric ===
              metric.key,
          }"
          @click="
            selectMetric(
              metric.key,
            )
          "
        >
          {{
            metric.shortLabel
          }}
        </button>
      </div>

      <div
        v-if="
          loading &&
          trendRows.length ===
            0
        "
        class="chart-loading"
      >
        <span class="spinner"></span>

        <p>
          Menyiapkan grafik transaksi...
        </p>
      </div>

      <div
        v-else
        class="chart-shell"
      >
        <svg
          class="market-chart"
          :viewBox="
            `0 0 ${chartWidth} ${chartHeight}`
          "
          preserveAspectRatio="none"
          @pointermove="
            handleChartPointerMove
          "
          @pointerleave="
            clearChartHover
          "
        >
          <defs>
            <linearGradient
              id="finance-area-gradient"
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
                stop-opacity="0.01"
              />
            </linearGradient>

            <filter
              id="finance-line-glow"
              x="-20%"
              y="-20%"
              width="140%"
              height="140%"
            >
              <feGaussianBlur
                stdDeviation="2.2"
                result="blur"
              />

              <feMerge>
                <feMergeNode
                  in="blur"
                />

                <feMergeNode
                  in="SourceGraphic"
                />
              </feMerge>
            </filter>
          </defs>

          <g
            class="grid-lines"
          >
            <g
              v-for="
                tick in yTicks
              "
              :key="
                tick.ratio
              "
            >
              <line
                :x1="
                  chartPadding.left
                "
                :x2="
                  chartWidth -
                  chartPadding.right
                "
                :y1="
                  tick.y
                "
                :y2="
                  tick.y
                "
              />

              <text
                x="8"
                :y="
                  tick.y + 4
                "
              >
                {{
                  formatMetricCompact(
                    tick.value,
                  )
                }}
              </text>
            </g>
          </g>

          <g
            class="volume-bars"
          >
            <rect
              v-for="(
                point,
                index
              ) in chartPoints"
              :key="
                `volume-${index}`
              "
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
                    point.invoice_count ||
                    0,
                  ) /
                  chartMaxInvoices
                ) *
                  34
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
                    point.invoice_count ||
                    0,
                  ) /
                  chartMaxInvoices
                ) *
                  34
              "
              rx="2"
            />
          </g>

          <path
            v-if="
              chartAreaPath
            "
            class="area-path"
            :d="
              chartAreaPath
            "
          />

          <path
            v-if="
              chartLinePath
            "
            class="line-path"
            :d="
              chartLinePath
            "
            filter="url(#finance-line-glow)"
          />

          <g
            class="point-layer"
          >
            <circle
              v-for="(
                point,
                index
              ) in chartPoints"
              :key="
                `point-${index}`
              "
              :cx="
                point.x
              "
              :cy="
                point.y
              "
              :r="
                hoveredIndex ===
                index
                  ? 5
                  : 2.6
              "
              :class="{
                hovered:
                  hoveredIndex ===
                  index,
              }"
            />
          </g>

          <g
            v-if="
              hoveredPoint
            "
            class="crosshair"
          >
            <line
              :x1="
                hoveredPoint.x
              "
              :x2="
                hoveredPoint.x
              "
              :y1="
                chartTop
              "
              :y2="
                chartBottom
              "
            />

            <line
              :x1="
                chartPadding.left
              "
              :x2="
                chartWidth -
                chartPadding.right
              "
              :y1="
                hoveredPoint.y
              "
              :y2="
                hoveredPoint.y
              "
            />

            <circle
              :cx="
                hoveredPoint.x
              "
              :cy="
                hoveredPoint.y
              "
              r="6.5"
            />
          </g>

          <g
            class="x-axis-labels"
          >
            <text
              v-for="
                label in xLabels
              "
              :key="
                label.index
              "
              :x="
                label.x
              "
              :y="
                chartHeight - 20
              "
              text-anchor="middle"
            >
              {{
                label.label
              }}
            </text>
          </g>

          <rect
            :x="
              chartPadding.left
            "
            :y="
              chartTop
            "
            :width="
              chartPlotWidth
            "
            :height="
              chartPlotHeight
            "
            fill="transparent"
          />
        </svg>

        <div
          v-if="
            hoveredPoint
          "
          class="chart-tooltip"
          :style="{
            left:
              `${Math.min(
                88,
                Math.max(
                  8,
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
            {{
              formatPeriod(
                hoveredPoint.period_start,
              )
            }}
          </span>

          <strong>
            {{
              formatMetricValue(
                hoveredPoint.metricValue,
              )
            }}
          </strong>

          <div>
            <small>
              Faktur
            </small>

            <b>
              {{
                formatNumber(
                  hoveredPoint.invoice_count,
                )
              }}
            </b>
          </div>

          <div>
            <small>
              Customer
            </small>

            <b>
              {{
                formatNumber(
                  hoveredPoint.customer_count,
                )
              }}
            </b>
          </div>

          <div>
            <small>
              Qty
            </small>

            <b>
              {{
                formatNumber(
                  hoveredPoint.total_qty,
                )
              }}
            </b>
          </div>
        </div>

        <div
          v-if="
            chartData.length ===
            0
          "
          class="chart-empty"
        >
          Belum ada data transaksi pada periode ini.
        </div>
      </div>

      <footer class="market-footer">
        <div>
          <span>
            Titik Tertinggi
          </span>

          <strong>
            {{
              highestPoint
                ? formatMetricValue(
                    highestPoint.metricValue,
                  )
                : formatMetricValue(
                    0,
                  )
            }}
          </strong>

          <small>
            {{
              highestPoint
                ? formatPeriod(
                    highestPoint.period_start,
                  )
                : '—'
            }}
          </small>
        </div>

        <div>
          <span>
            Rata-rata Per Titik
          </span>

          <strong>
            {{
              formatMetricValue(
                averageMetric,
              )
            }}
          </strong>

          <small>
            {{
              granularityLabel
            }}
          </small>
        </div>

        <div>
          <span>
            Penjualan Bersih
          </span>

          <strong>
            {{
              formatCurrency(
                summary.net_sales,
              )
            }}
          </strong>

          <small>
            setelah diskon sebelum pajak
          </small>
        </div>

        <div>
          <span>
            Total Diskon
          </span>

          <strong>
            {{
              formatCurrency(
                summary.discount_amount,
              )
            }}
          </strong>

          <small>
            seluruh potongan periode
          </small>
        </div>
      </footer>
    </section>

    <section class="transaction-panel">
      <header>
        <div>
          <span>
            TRANSACTION MIX
          </span>

          <h2>
            Komposisi Transaksi
          </h2>
        </div>

        <small>
          {{ periodLabel }}
        </small>
      </header>

      <div class="mix-grid">
        <article>
          <div class="mix-top">
            <div>
              <span>
                Penjualan Tempo
              </span>

              <strong>
                {{
                  formatNumber(
                    summary.term_count,
                  )
                }}
                faktur
              </strong>
            </div>

            <b>
              {{ termPercent }}%
            </b>
          </div>

          <div class="mix-track">
            <i
              :style="{
                width:
                  `${termPercent}%`,
              }"
            ></i>
          </div>
        </article>

        <article>
          <div class="mix-top">
            <div>
              <span>
                Pembayaran Langsung
              </span>

              <strong>
                {{
                  formatNumber(
                    summary.immediate_count,
                  )
                }}
                faktur
              </strong>
            </div>

            <b>
              {{ immediatePercent }}%
            </b>
          </div>

          <div class="mix-track">
            <i
              :style="{
                width:
                  `${immediatePercent}%`,
              }"
            ></i>
          </div>
        </article>

        <article>
          <div class="mix-top">
            <div>
              <span>
                Faktur Tercetak
              </span>

              <strong>
                {{
                  formatNumber(
                    summary.printed_count,
                  )
                }}
                faktur
              </strong>
            </div>

            <b>
              {{ printedPercent }}%
            </b>
          </div>

          <div class="mix-track">
            <i
              :style="{
                width:
                  `${printedPercent}%`,
              }"
            ></i>
          </div>
        </article>
      </div>
    </section>
        <section class="product-trend" aria-labelledby="product-trend-heading">
      <header>
        <div><h2 id="product-trend-heading">Tren Produk untuk Pembelian</h2><p>Lihat nama barang yang laris, laku sedang, dan belum terjual sebagai bahan pertimbangan pembelian ke supplier.</p><strong class="product-period-label">{{ productTrendPeriodLabel }}</strong></div>
        <div class="product-tools"><button :disabled="productTrendLoading || productTrendExporting" @click="productTrendLoad()">Perbarui</button><button :disabled="!productTrendCanReport" @click="productTrendExport('excel')">Excel Produk</button><button :disabled="!productTrendCanReport" @click="productTrendExport('print')">Print / PDF Produk</button></div>
      </header>
      <div class="product-period-controls">
        <div class="product-period-presets"><button :class="{ active: activePreset === 'THIS_MONTH' }" :disabled="loading || productTrendExporting" @click="applyPreset('THIS_MONTH')">Bulan Ini</button><button :class="{ active: activePreset === 'SIX_MONTHS' }" :disabled="loading || productTrendExporting" @click="applyPreset('SIX_MONTHS')">6 Bulan</button><button :class="{ active: activePreset === 'TWELVE_MONTHS' }" :disabled="loading || productTrendExporting" @click="applyPreset('TWELVE_MONTHS')">1 Tahun</button></div>
        <div class="product-month-control"><label><span>Bulan tertentu</span><input v-model="productTrendMonth" type="month" :max="todayISO().slice(0, 7)" /></label><button :disabled="loading || productTrendExporting" @click="productTrendApplyMonth">Tampilkan Bulan</button></div>
        <div class="product-manual-period"><label><span>Tanggal manual dari</span><input v-model="filters.date_from" type="date" @change="activePreset = 'CUSTOM'" /></label><label><span>Sampai</span><input v-model="filters.date_to" type="date" @change="activePreset = 'CUSTOM'" /></label><button :disabled="loading || productTrendExporting" @click="applyCustomPeriod">Terapkan Periode</button></div>
      </div>
      <div class="product-filter-row"><label class="product-search"><span>Cari nama barang / kode / SKU</span><input v-model="productTrendQuery" type="search" maxlength="200" placeholder="Contoh: nama obat atau kode barang" /></label><label class="product-sort"><span>Urutkan dalam setiap kelompok</span><FasSelect v-model="productTrendSort" aria-label="Urutan tren produk"><option v-for="[key, text] in productTrendSortOptions" :key="key" :value="key">{{ text }}</option></FasSelect></label></div>
      <div class="category-tabs" aria-label="Kelompok penjualan produk"><button v-for="[key, text] in productTrendCategories" :key="key" :class="{ active: productTrendCategory === key }" :aria-pressed="productTrendCategory === key" @click="productTrendCategory = key">{{ text }} <span>{{ productTrendCount(key === 'ALL' ? Object.values(productTrendSummary).reduce((n, value) => n + Number(value), 0) : productTrendSummary[key]) }}</span></button></div>
      <p class="explanation">Laris: frekuensi faktur minimal 50% dari produk paling sering dibeli. Laku sedang: sudah terjual dengan frekuensi di bawah batas itu. Belum terjual: nol penjualan dalam periode pilihan. Stok kini dan harga beli membantu pertimbangan pembelian; keduanya bukan riwayat stok periode tersebut. Faktur batal dikecualikan; retur belum diperhitungkan.</p>
      <p v-if="productTrendError" class="error" role="alert">{{ productTrendError }}</p>
      <p v-if="productTrendReportError" class="error" role="alert">{{ productTrendReportError }}</p>
      <div class="table-scroll" :aria-busy="productTrendLoading"><table><thead><tr><th>Peringkat</th><th class="product-name-cell">Nama Produk / Kode</th><th>Kelompok</th><th class="numeric">Frekuensi Faktur</th><th class="numeric">Jumlah Terjual</th><th class="numeric">Stok Kini</th><th class="numeric">Harga Beli / Unit</th><th class="numeric">Nilai Penjualan</th><th>Terakhir Terjual</th></tr></thead><tbody>
        <tr v-for="row in productTrendRows" :key="row.id"><td>#{{ productTrendCount(row.category_rank) }}</td><td class="product-name-cell"><strong>{{ row.name }}</strong><small>{{ row.product_code }} · {{ row.sku }}{{ row.is_active ? '' : ' · Nonaktif' }}</small></td><td><span class="badge">{{ productTrendLabel(row.category) }}</span></td><td class="numeric">{{ productTrendCount(row.invoice_count) }}</td><td class="numeric">{{ productTrendCount(row.qty) }} {{ row.unit }}</td><td class="numeric"><strong :class="{ 'stock-low': Number(row.stock) <= Number(row.min_stock) }">{{ productTrendCount(row.stock) }} {{ row.unit }}</strong><small>Minimum {{ productTrendCount(row.min_stock) }}</small></td><td class="numeric">{{ productTrendMoney(row.purchase_price) }}</td><td class="numeric">{{ productTrendMoney(row.net_sales) }}</td><td>{{ productTrendDate(row.last_sold) }}</td></tr>
        <tr v-if="!productTrendRows.length"><td colspan="9" class="empty">{{ productTrendLoading ? 'Memuat nama dan peringkat produk…' : productTrendError ? 'Daftar produk belum dapat dimuat.' : 'Tidak ada produk yang sesuai periode dan filter.' }}</td></tr>
      </tbody></table></div>
      <footer><span>{{ productTrendRows.length ? productTrendOffset + 1 : 0 }}–{{ productTrendOffset + productTrendRows.length }} dari {{ productTrendCount(productTrendTotal) }} produk · Halaman {{ productTrendPage }} dari {{ productTrendPages }} · Batas {{ productTrendLimit }} produk per halaman. Peringkat dihitung dalam kelompok yang sama.</span><div><label class="page-size"><span>Produk / halaman</span><FasSelect v-model="productTrendLimit" :disabled="productTrendLoading || productTrendExporting" aria-label="Jumlah produk per halaman"><option :value="10">10 produk</option><option :value="15">15 produk</option><option :value="25">25 produk</option></FasSelect></label><button :disabled="productTrendLoading || productTrendExporting || productTrendPage <= 1" @click="productTrendChangePage(-1)">Sebelumnya</button><button :disabled="productTrendLoading || productTrendExporting || productTrendPage >= productTrendPages" @click="productTrendChangePage(1)">Berikutnya</button></div></footer>
    </section>

  </div>
</template>

<style scoped>
.analytics-page {
  min-height:
    100%;
  padding:
    1.25rem
    1.5rem
    2.5rem;
  color:
    var(--fas-text);
}

.analytics-header {
  display:
    flex;
  align-items:
    flex-end;
  justify-content:
    space-between;
  gap:
    1rem;
}

.analytics-header h1 {
  margin:
    0.45rem
    0
    0;
  color:
    var(--fas-text);
  font-size:
    1.55rem;
  font-weight:
    800;
  letter-spacing:
    -0.035em;
}

.analytics-header p {
  max-width:
    52rem;
  margin:
    0.38rem
    0
    0;
  color:
    var(--fas-muted);
  font-size:
    0.62rem;
  line-height:
    1.05rem;
}

.live-status {
  display:
    flex;
  align-items:
    center;
  gap:
    0.38rem;
  color:
    var(--fas-muted);
  font-size:
    0.5rem;
  font-weight:
    750;
  letter-spacing:
    0.08em;
}

.live-dot {
  width:
    0.48rem;
  height:
    0.48rem;
  border-radius:
    999px;
  background:
    #10b981;
  box-shadow:
    0
    0
    0.8rem
    rgba(
      16,
      185,
      129,
      0.55
    );
  animation:
    livePulse
    1.8s
    infinite;
}

.live-divider {
  color:
    #334155;
}

.header-actions {
  display:
    flex;
  flex-wrap:
    wrap;
  justify-content:
    flex-end;
  gap:
    0.45rem;
}

.action-button,
.apply-button {
  min-height:
    2.45rem;
  border-radius:
    0.65rem;
  padding:
    0
    0.78rem;
  font-size:
    0.55rem;
  font-weight:
    800;
  cursor:
    pointer;
}

.action-button.secondary,
.apply-button {
  border:
    1px solid
    var(--fas-border);
  background:
    var(--fas-card);
  color:
    var(--fas-text-soft);
}

.action-button.primary {
  border:
    1px solid
    var(--fas-accent-border);
  background:
    var(--fas-accent);
  color:
    #ffffff;
}

.action-button:disabled {
  cursor:
    not-allowed;
  opacity:
    0.5;
}

.finance-nav {
  display:
    flex;
  flex-wrap:
    wrap;
  gap:
    0.45rem;
  margin-top:
    1.05rem;
}

.finance-nav a {
  display:
    inline-flex;
  min-height:
    2.1rem;
  align-items:
    center;
  justify-content:
    center;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.58rem;
  background:
    var(--fas-card);
  padding:
    0
    0.68rem;
  color:
    var(--fas-muted);
  font-size:
    0.5rem;
  font-weight:
    800;
  text-decoration:
    none;
}

.finance-nav a.active {
  border-color:
    var(--fas-accent-border);
  background:
    var(--fas-accent-soft);
  color:
    var(--fas-accent-text);
}

.message-error {
  margin-top:
    0.9rem;
  border:
    1px solid
    rgba(
      248,
      113,
      113,
      0.25
    );
  border-radius:
    0.7rem;
  background:
    rgba(
      127,
      29,
      29,
      0.15
    );
  padding:
    0.7rem
    0.85rem;
  color:
    #fca5a5;
  font-size:
    0.58rem;
  font-weight:
    700;
}

.period-panel {
  display:
    flex;
  align-items:
    flex-end;
  justify-content:
    space-between;
  gap:
    0.85rem;
  margin-top:
    1rem;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.95rem;
  background:
    rgba(
      9,
      16,
      29,
      0.72
    );
  padding:
    0.7rem;
}

.preset-list {
  display:
    flex;
  flex-wrap:
    wrap;
  gap:
    0.38rem;
}

.preset-list button {
  min-height:
    2.2rem;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.55rem;
  background:
    var(--fas-card);
  padding:
    0
    0.65rem;
  color:
    var(--fas-muted);
  font-size:
    0.5rem;
  font-weight:
    800;
  cursor:
    pointer;
}

.preset-list button.active {
  border-color:
    var(--fas-accent-border);
  background:
    var(--fas-accent-soft);
  color:
    var(--fas-accent-text);
}

.custom-period {
  display:
    flex;
  align-items:
    flex-end;
  gap:
    0.4rem;
}

.custom-period label {
  display:
    grid;
  gap:
    0.22rem;
}

.custom-period label span {
  color:
    var(--fas-muted);
  font-size:
    0.45rem;
  font-weight:
    750;
}

.custom-period input {
  min-height:
    2.2rem;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.55rem;
  outline:
    none;
  background:
    #070e1a;
  padding:
    0
    0.55rem;
  color:
    var(--fas-text-soft);
  font-size:
    0.52rem;
  color-scheme:
    inherit;
}

.arrow {
  display:
    flex;
  min-height:
    2.2rem;
  align-items:
    center;
  color:
    var(--fas-muted-soft);
}

.summary-grid {
  display:
    grid;
  grid-template-columns:
    repeat(
      4,
      minmax(
        0,
        1fr
      )
    );
  gap:
    0.72rem;
  margin-top:
    0.9rem;
}

.summary-card {
  display:
    flex;
  min-height:
    8.5rem;
  flex-direction:
    column;
  justify-content:
    space-between;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.92rem;
  background:
    linear-gradient(
      145deg,
      var(--fas-card),
      var(--fas-card)
    );
  padding:
    0.85rem;
}

.summary-card.featured {
  border-color:
    var(--fas-accent-border);
  background:
    radial-gradient(
      circle at 90% 10%,
      var(--fas-accent-soft),
      transparent 46%
    ),
    linear-gradient(
      145deg,
      var(--fas-card),
      var(--fas-card)
    );
}

.summary-card span {
  color:
    var(--fas-muted);
  font-size:
    0.55rem;
  font-weight:
    750;
}

.summary-card small {
  display:
    block;
  margin-top:
    0.22rem;
  color:
    var(--fas-muted-soft);
  font-size:
    0.44rem;
}

.summary-card > strong {
  color:
    var(--fas-text);
  font-size:
    1.35rem;
  font-weight:
    850;
  letter-spacing:
    -0.04em;
}

.summary-card p {
  margin:
    0;
  color:
    var(--fas-muted);
  font-size:
    0.5rem;
}

.market-panel,
.transaction-panel {
  margin-top:
    0.9rem;
  overflow:
    hidden;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    1rem;
  background:
    var(--fas-card);
}

.market-header {
  display:
    flex;
  align-items:
    flex-start;
  justify-content:
    space-between;
  gap:
    1rem;
  border-bottom:
    1px solid
    var(--fas-border);
  padding:
    0.95rem
    1rem
    0.8rem;
}

.market-kicker {
  color:
    var(--fas-accent-text);
  font-size:
    0.48rem;
  font-weight:
    850;
  letter-spacing:
    0.1em;
}

.market-header h2 {
  margin:
    0.28rem
    0
    0;
  color:
    var(--fas-text);
  font-size:
    0.95rem;
  font-weight:
    800;
}

.market-value-row {
  display:
    flex;
  align-items:
    center;
  gap:
    0.55rem;
  margin-top:
    0.42rem;
}

.market-value-row > strong {
  color:
    var(--fas-text);
  font-size:
    1.25rem;
  letter-spacing:
    -0.035em;
}

.change-badge {
  display:
    inline-flex;
  min-height:
    1.5rem;
  align-items:
    center;
  gap:
    0.22rem;
  border-radius:
    999px;
  padding:
    0
    0.46rem;
  font-size:
    0.46rem;
  font-weight:
    850;
}

.change-up {
  background:
    rgba(
      16,
      185,
      129,
      0.1
    );
  color:
    #6ee7b7;
}

.change-down {
  background:
    rgba(
      239,
      68,
      68,
      0.1
    );
  color:
    #fca5a5;
}

.change-flat {
  background:
    rgba(
      100,
      116,
      139,
      0.12
    );
  color:
    var(--fas-muted);
}

.market-header p {
  margin:
    0.25rem
    0
    0;
  color:
    var(--fas-muted);
  font-size:
    0.48rem;
}

.market-meta {
  display:
    flex;
  flex-wrap:
    wrap;
  justify-content:
    flex-end;
  gap:
    0.4rem;
  max-width:
    32rem;
}

.market-meta > span {
  display:
    inline-flex;
  min-height:
    1.75rem;
  align-items:
    center;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    999px;
  background:
    var(--fas-card);
  padding:
    0
    0.55rem;
  color:
    var(--fas-muted);
  font-size:
    0.45rem;
  font-weight:
    750;
}

.market-meta .live-pill {
  border-color:
    rgba(
      16,
      185,
      129,
      0.25
    );
  color:
    #6ee7b7;
}

.live-pill i {
  width:
    0.38rem;
  height:
    0.38rem;
  margin-right:
    0.28rem;
  border-radius:
    999px;
  background:
    #10b981;
}

.metric-tabs {
  display:
    flex;
  flex-wrap:
    wrap;
  gap:
    0.35rem;
  padding:
    0.7rem
    1rem
    0;
}

.metric-tabs button {
  min-height:
    1.95rem;
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.52rem;
  background:
    transparent;
  padding:
    0
    0.58rem;
  color:
    var(--fas-muted);
  font-size:
    0.47rem;
  font-weight:
    800;
  cursor:
    pointer;
}

.metric-tabs button.active {
  border-color:
    var(--fas-accent-border);
  background:
    var(--fas-accent-soft);
  color:
    var(--fas-accent-text);
}

.chart-shell {
  position:
    relative;
  min-height:
    25rem;
  padding:
    0.55rem
    0.75rem
    0;
}

.market-chart {
  width:
    100%;
  height:
    23rem;
  overflow:
    visible;
  touch-action:
    none;
}

.grid-lines line {
  stroke:
    var(--fas-border);
  stroke-width:
    1;
  vector-effect:
    non-scaling-stroke;
}

.grid-lines text,
.x-axis-labels text {
  fill:
    var(--fas-muted);
  font-size:
    11px;
  font-family:
    Inter,
    system-ui,
    sans-serif;
}

.volume-bars rect {
  fill:
    var(--fas-accent-soft);
}

.area-path {
  fill:
    url(
      #finance-area-gradient
    );
}

.line-path {
  fill:
    none;
  stroke:
    var(--fas-accent-text);
  stroke-width:
    2.3;
  stroke-linecap:
    round;
  stroke-linejoin:
    round;
  vector-effect:
    non-scaling-stroke;
}

.point-layer circle {
  fill:
    var(--fas-card);
  stroke:
    var(--fas-accent-text);
  stroke-width:
    1.8;
  vector-effect:
    non-scaling-stroke;
}

.point-layer circle.hovered {
  fill:
    var(--fas-accent-text);
  stroke:
    var(--fas-text);
}

.crosshair line {
  stroke:
    rgba(
      148,
      163,
      184,
      0.35
    );
  stroke-width:
    1;
  stroke-dasharray:
    4
    4;
  vector-effect:
    non-scaling-stroke;
}

.crosshair circle {
  fill:
    var(--fas-accent-text);
  stroke:
    var(--fas-text);
  stroke-width:
    2.2;
  vector-effect:
    non-scaling-stroke;
}

.chart-tooltip {
  position:
    absolute;
  top:
    1.2rem;
  z-index:
    5;
  min-width:
    10.5rem;
  transform:
    translateX(
      -50%
    );
  border:
    1px solid
    rgba(
      71,
      85,
      105,
      0.8
    );
  border-radius:
    0.7rem;
  background:
    var(--fas-card);
  box-shadow:
    0
    18px
    50px
    rgba(
      0,
      0,
      0,
      0.35
    );
  padding:
    0.65rem;
  pointer-events:
    none;
}

.chart-tooltip > span {
  color:
    var(--fas-muted);
  font-size:
    0.45rem;
}

.chart-tooltip > strong {
  display:
    block;
  margin-top:
    0.18rem;
  color:
    var(--fas-text);
  font-size:
    0.72rem;
}

.chart-tooltip > div {
  display:
    flex;
  justify-content:
    space-between;
  gap:
    1rem;
  margin-top:
    0.34rem;
  border-top:
    1px solid
    var(--fas-border);
  padding-top:
    0.28rem;
}

.chart-tooltip small {
  color:
    var(--fas-muted);
  font-size:
    0.42rem;
}

.chart-tooltip b {
  color:
    var(--fas-text-soft);
  font-size:
    0.45rem;
}

.chart-empty,
.chart-loading {
  display:
    grid;
  min-height:
    23rem;
  place-items:
    center;
  align-content:
    center;
  gap:
    0.6rem;
  color:
    var(--fas-muted);
  font-size:
    0.58rem;
}

.spinner {
  width:
    1.4rem;
  height:
    1.4rem;
  border:
    2px solid
    var(--fas-border);
  border-top-color:
    var(--fas-accent-text);
  border-radius:
    999px;
  animation:
    spin
    0.8s
    linear
    infinite;
}

.market-footer {
  display:
    grid;
  grid-template-columns:
    repeat(
      4,
      minmax(
        0,
        1fr
      )
    );
  border-top:
    1px solid
    var(--fas-border);
}

.market-footer > div {
  min-height:
    5.2rem;
  padding:
    0.72rem
    0.9rem;
}

.market-footer > div + div {
  border-left:
    1px solid
    var(--fas-border);
}

.market-footer span,
.market-footer small {
  display:
    block;
  color:
    var(--fas-muted);
  font-size:
    0.45rem;
}

.market-footer strong {
  display:
    block;
  margin:
    0.33rem
    0;
  color:
    var(--fas-text);
  font-size:
    0.65rem;
}

.transaction-panel > header {
  display:
    flex;
  align-items:
    center;
  justify-content:
    space-between;
  gap:
    1rem;
  border-bottom:
    1px solid
    var(--fas-border);
  padding:
    0.8rem
    1rem;
}

.transaction-panel > header span {
  color:
    var(--fas-accent-text);
  font-size:
    0.46rem;
  font-weight:
    850;
  letter-spacing:
    0.09em;
}

.transaction-panel > header h2 {
  margin:
    0.22rem
    0
    0;
  color:
    var(--fas-text);
  font-size:
    0.82rem;
}

.transaction-panel > header small {
  color:
    var(--fas-muted);
  font-size:
    0.48rem;
}

.mix-grid {
  display:
    grid;
  grid-template-columns:
    repeat(
      3,
      minmax(
        0,
        1fr
      )
    );
  gap:
    0.7rem;
  padding:
    0.85rem
    1rem;
}

.mix-grid article {
  border:
    1px solid
    var(--fas-border);
  border-radius:
    0.72rem;
  background:
    var(--fas-card);
  padding:
    0.7rem;
}

.mix-top {
  display:
    flex;
  align-items:
    center;
  justify-content:
    space-between;
  gap:
    0.75rem;
}

.mix-top span {
  color:
    var(--fas-muted);
  font-size:
    0.48rem;
}

.mix-top strong {
  display:
    block;
  margin-top:
    0.15rem;
  color:
    var(--fas-text-soft);
  font-size:
    0.55rem;
}

.mix-top > b {
  color:
    var(--fas-accent-text);
  font-size:
    0.58rem;
}

.mix-track {
  height:
    0.38rem;
  margin-top:
    0.55rem;
  overflow:
    hidden;
  border-radius:
    999px;
  background:
    var(--fas-elevated);
}

.mix-track i {
  display:
    block;
  height:
    100%;
  border-radius:
    inherit;
  background:
    linear-gradient(
      90deg,
      var(--fas-accent),
      var(--fas-accent-text)
    );
}

@keyframes livePulse {
  0%,
  100% {
    opacity:
      1;
    transform:
      scale(1);
  }

  50% {
    opacity:
      0.45;
    transform:
      scale(0.82);
  }
}

@keyframes spin {
  to {
    transform:
      rotate(
        360deg
      );
  }
}

@media (
  max-width:
  1180px
) {
  .summary-grid,
  .market-footer {
    grid-template-columns:
      repeat(
        2,
        minmax(
          0,
          1fr
        )
      );
  }

  .market-footer
  > div:nth-child(3) {
    border-left:
      0;
    border-top:
      1px solid
      var(--fas-border);
  }

  .market-footer
  > div:nth-child(4) {
    border-top:
      1px solid
      var(--fas-border);
  }

  .period-panel {
    align-items:
      stretch;
    flex-direction:
      column;
  }

  .custom-period {
    justify-content:
      flex-start;
  }
}

@media (
  max-width:
  820px
) {
  .analytics-page {
    padding:
      1rem
      0.8rem
      2rem;
  }

  .analytics-header,
  .market-header,
  .transaction-panel
  > header {
    align-items:
      stretch;
    flex-direction:
      column;
  }

  .header-actions,
  .market-meta {
    justify-content:
      flex-start;
  }

  .summary-grid,
  .mix-grid,
  .market-footer {
    grid-template-columns:
      1fr;
  }

  .market-footer
  > div + div {
    border-left:
      0;
    border-top:
      1px solid
      var(--fas-border);
  }

  .custom-period {
    align-items:
      stretch;
    flex-direction:
      column;
  }

  .arrow {
    display:
      none;
  }

  .market-chart {
    min-width:
      50rem;
  }

  .chart-shell {
    overflow-x:
      auto;
  }
}

.analytics-header h1 {
  font-size: 1.8rem;
}

.analytics-header > div > p {
  font-size: 0.85rem;
  line-height: 1.6;
}

.action-button,
.apply-button {
  min-height: 2.85rem;
  gap: 0.45rem;
  border-radius: 0.85rem;
  padding: 0.6rem 0.85rem;
  font-size: 0.8rem;
  font-weight: 700;
  line-height: 1.4;
}

.action-button:focus:not(:focus-visible),
.apply-button:focus:not(:focus-visible) {
  outline: none;
}

.action-button:focus-visible,
.apply-button:focus-visible,
.preset-list button:focus-visible,
.metric-tabs button:focus-visible {
  outline: 2px solid var(--fas-accent);
  outline-offset: 2px;
}

.period-panel {
  flex-wrap: wrap;
  gap: 1rem;
  background: var(--fas-card);
  padding: 1rem;
}

.preset-list {
  gap: 0.55rem;
}

.preset-list button {
  min-height: 2.85rem;
  border-radius: 0.85rem;
  padding: 0.55rem 0.8rem;
  font-size: 0.75rem;
  font-weight: 700;
}

.custom-period {
  flex-wrap: wrap;
  gap: 0.65rem;
}

.custom-period label {
  min-width: 9rem;
  flex: 1;
  gap: 0.4rem;
}

.custom-period label span {
  color: var(--fas-text-soft);
  font-size: 0.72rem;
}

.custom-period input {
  width: 100%;
  min-height: 2.85rem;
  border-color: var(--fas-border-strong);
  border-radius: 0.85rem;
  background: var(--fas-input);
  padding: 0.6rem 0.8rem;
  color: var(--fas-text);
  font-size: 0.8rem;
}

.custom-period input:focus {
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.metric-tabs {
  flex-wrap: wrap;
  gap: 0.5rem;
}

.metric-tabs button {
  min-height: 2.5rem;
  border-radius: 0.75rem;
  padding: 0.5rem 0.75rem;
  font-size: 0.75rem;
}

.message-error {
  border-color: var(--fas-danger);
  background: var(--fas-danger-soft);
  color: var(--fas-danger);
  font-size: 0.8rem;
  line-height: 1.5;
}

@media (max-width: 640px) {
  .custom-period {
    width: 100%;
  }

  .custom-period label {
    min-width: 0;
  }

  .analytics-header h1 {
    font-size: 1.55rem;
  }
}
.product-trend .table-scroll{max-height:min(65vh,42rem)}.product-trend thead{position:sticky;top:0;z-index:1}.product-trend footer div{align-items:center;flex-wrap:wrap}.product-trend .page-size{display:flex;flex-direction:column;gap:.3rem;min-width:9rem}.product-trend .page-size>span{font-size:.7rem}
</style>

<style scoped>

.product-trend{margin-top:1rem;border:1px solid var(--fas-border);border-radius:1.1rem;background:var(--fas-card);color:var(--fas-text);overflow:hidden;min-width:0}
.product-trend header{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:1rem;padding:1.1rem}.product-trend h2{font-size:1.05rem;font-weight:800}.product-trend header p,.product-trend .explanation{font-size:.78rem;color:var(--fas-muted);line-height:1.6;margin-top:.4rem}.product-trend header label{flex:0 1 23rem;min-width:0}.product-trend input{width:100%;min-height:2.85rem;border:1px solid var(--fas-border-strong);border-radius:.85rem;background:var(--fas-input);color:var(--fas-text);padding:.6rem .85rem;font-size:.8rem}
.product-trend .category-tabs{display:flex;gap:.5rem;flex-wrap:wrap;padding:0 1.1rem}.product-trend .category-tabs button,.product-trend footer button{min-height:2.85rem;border-radius:.85rem;padding:.6rem .85rem;border:1px solid var(--fas-border);background:var(--fas-elevated);color:var(--fas-text-soft);font-size:.8rem;cursor:pointer}.product-trend .category-tabs .active{background:var(--fas-accent-soft);border-color:var(--fas-accent-border);color:var(--fas-accent-text)}.product-trend .category-tabs span{margin-left:.45rem;font-weight:700}.product-trend .explanation{padding:.7rem 1.1rem;margin:0}.product-trend .error{margin:.5rem 1.1rem;padding:.8rem;color:var(--fas-danger);background:var(--fas-danger-soft);border-radius:.85rem}
.product-trend .table-scroll{overflow:auto}.product-trend table{width:100%;min-width:850px;border-collapse:collapse;font-size:.8rem}.product-trend th,.product-trend td{text-align:left;padding:.85rem 1.1rem;border-top:1px solid var(--fas-border)}.product-trend th{font-size:.73rem;color:var(--fas-muted);background:var(--fas-elevated)}.product-trend td strong,.product-trend td small{display:block}.product-trend td small{color:var(--fas-muted);font-size:.7rem;margin-top:.3rem}.product-trend .numeric{text-align:right;white-space:nowrap}.product-trend .badge{display:inline-block;padding:.3rem .6rem;border-radius:.6rem;background:var(--fas-accent-soft);color:var(--fas-accent-text);font-size:.7rem}.product-trend .empty{text-align:center;padding:2rem;color:var(--fas-muted)}.product-trend footer{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:.75rem;padding:1rem 1.1rem;border-top:1px solid var(--fas-border);font-size:.75rem;color:var(--fas-muted)}.product-trend footer div{display:flex;gap:.5rem}.product-trend button:disabled{opacity:.5;cursor:not-allowed}.product-trend button:focus-visible,.product-trend input:focus-visible{outline:2px solid var(--fas-accent);outline-offset:2px}.product-trend .sr-only{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0}

.product-trend .table-scroll{max-height:min(65vh,42rem)}.product-trend thead{position:sticky;top:0;z-index:1}.product-trend footer div{align-items:center;flex-wrap:wrap}.product-trend .page-size{display:flex;flex-direction:column;gap:.3rem;min-width:9rem}.product-trend .page-size>span{font-size:.7rem}
</style>

<style scoped>
.product-trend header{align-items:flex-start}
.product-trend header > div:first-child{flex:1 1 26rem}
.product-trend .product-period-label{display:block;margin-top:.65rem;font-size:.82rem;color:var(--fas-accent-text)}
.product-trend .product-tools,.product-trend .product-period-presets,.product-trend .product-month-control,.product-trend .product-manual-period{display:flex;align-items:flex-end;gap:.6rem;flex-wrap:wrap}
.product-trend .product-tools button,.product-trend .product-period-controls button{min-height:2.85rem;border:1px solid var(--fas-border-strong);border-radius:.85rem;padding:.6rem .85rem;background:var(--fas-card);color:var(--fas-text-soft);font-size:.78rem;font-weight:700;cursor:pointer}
.product-trend .product-period-presets .active{color:var(--fas-accent-text);background:var(--fas-accent-soft);border-color:var(--fas-accent-border)}
.product-trend .product-period-controls{display:grid;gap:1rem;padding:1rem 1.1rem;background:var(--fas-elevated);border-top:1px solid var(--fas-border);border-bottom:1px solid var(--fas-border)}
.product-trend .product-period-controls label,.product-trend .product-filter-row label{display:flex;flex-direction:column;gap:.45rem;color:var(--fas-text-soft);font-size:.75rem;min-width:0}
.product-trend .product-month-control label{min-width:13rem;flex:0 1 16rem}
.product-trend .product-manual-period label{flex:0 1 16rem;min-width:12rem}
.product-trend .product-filter-row{display:grid;grid-template-columns:minmax(0,1.5fr) minmax(14rem,1fr);gap:.85rem;padding:1.1rem}
.product-trend .product-period-controls input,.product-trend .product-filter-row input{min-height:2.85rem}
.product-trend table{min-width:1200px}
.product-trend .product-name-cell{min-width:15rem;max-width:22rem;white-space:normal;overflow-wrap:anywhere}
.product-trend .stock-low{color:var(--fas-danger)}
.product-trend button:disabled{opacity:.5;cursor:not-allowed}
@media(max-width:700px){.product-trend .product-filter-row{grid-template-columns:1fr}.product-trend .product-month-control label,.product-trend .product-manual-period label{flex:1 1 100%;min-width:0}.product-trend .product-tools{width:100%}.product-trend .product-tools button{flex:1}}

.product-trend .table-scroll{max-height:min(65vh,42rem)}.product-trend thead{position:sticky;top:0;z-index:1}.product-trend footer div{align-items:center;flex-wrap:wrap}.product-trend .page-size{display:flex;flex-direction:column;gap:.3rem;min-width:9rem}.product-trend .page-size>span{font-size:.7rem}
</style>
