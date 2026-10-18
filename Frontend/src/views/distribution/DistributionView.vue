<script setup>
import { companyProfile } from '@/services/company'

import FasIcon from '@/components/ui/FasIcon.vue'
import FasSelect from '@/components/ui/FasSelect.vue'

import {
  computed,
  nextTick,
  onMounted,
  ref,
} from 'vue'

import { useAuthStore } from '@/stores/auth'
import {
  getSalesOrders,
  getSalesOrderByID,
  shipSalesOrder,
} from '@/services/salesOrder'

import QRCode from 'qrcode'

import {
  ensureInvoiceBySalesOrder,
  updateInvoice,
  markInvoicePrinted,
} from '@/services/invoice'

import {
  getWarehouses,
} from '@/services/warehouse'

import {
  getTaxProfiles,
  getCurrentTaxProfile,
  saveStandardTaxProfile,
} from '@/services/tax'

const authStore = useAuthStore()

const loading = ref(true)
const errorMessage = ref('')
const successMessage = ref('')
const orders = ref([])

const search = ref('')
const statusFilter = ref('ALL')
const priorityFilter = ref('ALL')
const expandedOrderID = ref('')
const loadingDetailID = ref('')
const shippingOrderID = ref('')
const invoiceLoadingOrderID = ref('')
const invoiceOpen = ref(false)
const invoiceData = ref(null)
const invoiceEditing = ref(false)
const invoiceSaving = ref(false)
const invoicePrinting = ref(false)
const invoiceQRCode = ref('')
const invoiceForm = ref(null)
const activeInvoiceOrderID = ref('')

const RECIPIENT_STORAGE_KEY = 'fas_distribution_invoice_recipients'

const loadRecipientOverrides = () => {
  try {
    const raw =
      window.localStorage.getItem(
        RECIPIENT_STORAGE_KEY,
      )

    if (!raw) {
      return {}
    }

    const parsed =
      JSON.parse(raw)

    return (
      parsed &&
      typeof parsed === 'object'
        ? parsed
        : {}
    )
  } catch {
    return {}
  }
}

const invoiceIssuer = computed(() => invoiceData.value?.issuer_profile || companyProfile)
const companyAddress = computed(() => invoiceIssuer.value.address || '')

const recipientOverrides = ref(
  loadRecipientOverrides(),
)

const recipientEditOrderID = ref('')

const recipientForm = ref({
  name: '',
  address: '',
})

const PRINT_PAPER_STORAGE_KEY =
  'fas_invoice_print_paper'

const getInitialPrintPaperMode = () => {
  try {
    const saved =
      window.localStorage.getItem(
        PRINT_PAPER_STORAGE_KEY,
      )

    if (
      saved === 'SALES' ||
      saved === 'A4'
    ) {
      return saved
    }
  } catch {
    return 'SALES'
  }

  return 'SALES'
}

const printPaperMode = ref(
  getInitialPrintPaperMode(),
)

const warehouses = ref([])
const warehouseLoading = ref(false)
const taxProfiles = ref([])
const currentTaxProfile = ref(null)
const taxLoading = ref(false)
const taxRuleEditorOpen = ref(false)
const taxRuleSaving = ref(false)
const taxRuleForm = ref({
  name: 'PPN Standar',
  nominal_rate: 12,
  dpp_factor_percent: 91.6667,
  regulation_reference: '',
  notes: '',
})

const DISTRIBUTION_HISTORY_DAYS =
  3

const DISTRIBUTION_HISTORY_MS =
  DISTRIBUTION_HISTORY_DAYS *
  24 *
  60 *
  60 *
  1000

const currentRole = computed(() => {
  return String(
    authStore.user?.role || '',
  ).toLowerCase()
})

const canManageDistribution = computed(() => {
  return [
    'admin',
    'owner',
    'administrator',
    'pegawai',
    'warehouse',
    'gudang',
  ].includes(currentRole.value)
})

const canManageTax = computed(() => {
  return [
    'admin',
    'owner',
    'administrator',
    'finance',
  ].includes(
    currentRole.value,
  )
})

const printPaperOptions = [
  {
    value: 'SALES',
    label: 'Faktur Penjualan',
    shortLabel: '9.5 × 5.5 inci',
    description:
      'Ukuran faktur distributor dengan maksimal 10 barang per lembar.',
    widthLabel: '241 × 140 mm',
    itemsPerPage: 10,
  },
  {
    value: 'A4',
    label: 'A4 Distributor',
    shortLabel: 'A4',
    description:
      'Ukuran A4 landscape untuk daftar barang lebih panjang.',
    widthLabel: '297 × 210 mm',
    itemsPerPage: 18,
  },
]

const currentPrintPaper = computed(() => {
  return (
    printPaperOptions.find(
      (paper) =>
        paper.value ===
        printPaperMode.value,
    ) ||
    printPaperOptions[0]
  )
})

const invoicePrintPages = computed(() => {
  const source =
    Array.isArray(
      invoiceData.value?.items,
    )
      ? invoiceData.value.items
      : []

  const itemsPerPage =
    Math.max(
      1,
      Number(
        currentPrintPaper.value
          ?.itemsPerPage ||
        10,
      ),
    )

  if (source.length === 0) {
    return [
      {
        number: 1,
        items: [],
        isFirst: true,
        isLast: true,
      },
    ]
  }

  const pages = []

  for (
    let startIndex = 0;
    startIndex < source.length;
    startIndex += itemsPerPage
  ) {
    const pageItems =
      source
        .slice(
          startIndex,
          startIndex +
            itemsPerPage,
        )
        .map(
          (item, index) => ({
            ...item,
            __originalIndex:
              startIndex +
              index,
          }),
        )

    pages.push(
      pageItems,
    )
  }

  return pages.map(
    (items, index) => ({
      number: index + 1,
      items,
      isFirst:
        index === 0,
      isLast:
        index ===
        pages.length - 1,
    }),
  )
})

const invoicePrintPageCount = computed(() => {
  return invoicePrintPages.value.length
})

const setPrintPaperMode = (mode) => {
  if (
    mode !== 'SALES' &&
    mode !== 'A4'
  ) {
    return
  }

  printPaperMode.value = mode

  try {
    window.localStorage.setItem(
      PRINT_PAPER_STORAGE_KEY,
      mode,
    )
  } catch {
    return
  }
}

const getDynamicPrintPageCSS = () => {
  if (
    printPaperMode.value ===
    'A4'
  ) {
    return `
      @page {
        size: A4 landscape;
        margin: 0;
      }
    `
  }

  return `
    @page {
      size: 9.5in 5.5in;
      margin: 0;
    }
  `
}

const installDynamicPrintPageStyle = () => {
  const styleID =
    'fas-dynamic-invoice-page-size'

  let style =
    document.getElementById(
      styleID,
    )

  if (!style) {
    style =
      document.createElement(
        'style',
      )

    style.id = styleID
    document.head.appendChild(
      style,
    )
  }

  style.textContent =
    getDynamicPrintPageCSS()
}

const removeDynamicPrintPageStyle = () => {
  const style =
    document.getElementById(
      'fas-dynamic-invoice-page-size',
    )

  if (style) {
    style.remove()
  }
}

const companyAddressLines = computed(() => String(companyAddress.value).split(/\r?\n/).map(line => line.trim()).filter(Boolean))

const getRecipientOverride = (
  orderID,
) => {
  if (!orderID) {
    return null
  }

  return (
    recipientOverrides.value?.[
      orderID
    ] ||
    null
  )
}

const getOrderRecipientName = (
  order,
) => {
  const override =
    getRecipientOverride(
      order?.id,
    )

  return (
    override?.name ||
    order?.customer_name ||
    '—'
  )
}

const getOrderRecipientAddress = (
  order,
) => {
  const override =
    getRecipientOverride(
      order?.id,
    )

  return (
    override?.address ||
    order?.shipping_address ||
    order?.customer_address ||
    ''
  )
}

const invoiceRecipientName = computed(() => {
  const override =
    getRecipientOverride(
      activeInvoiceOrderID.value,
    )

  return (
    override?.name ||
    invoiceData.value?.store_name ||
    '—'
  )
})

const invoiceRecipientAddress = computed(() => {
  const override =
    getRecipientOverride(
      activeInvoiceOrderID.value,
    )

  return (
    override?.address ||
    invoiceData.value
      ?.customer_tax_address ||
    invoiceData.value
      ?.shipping_address ||
    '—'
  )
})

const beginRecipientEdit = (
  order,
) => {
  if (!order?.id) {
    return
  }

  recipientEditOrderID.value =
    order.id

  recipientForm.value = {
    name:
      getOrderRecipientName(
        order,
      ) === '—'
        ? ''
        : getOrderRecipientName(
            order,
          ),
    address:
      getOrderRecipientAddress(
        order,
      ),
  }
}

const cancelRecipientEdit = () => {
  recipientEditOrderID.value = ''

  recipientForm.value = {
    name: '',
    address: '',
  }
}

const saveRecipientEdit = (
  order,
) => {
  if (!order?.id) {
    return
  }

  const name =
    String(
      recipientForm.value.name ||
      '',
    ).trim()

  const address =
    String(
      recipientForm.value.address ||
      '',
    ).trim()

  if (!name) {
    errorMessage.value =
      'Nama penerima faktur wajib diisi.'
    return
  }

  recipientOverrides.value = {
    ...recipientOverrides.value,
    [order.id]: {
      name,
      address,
    },
  }

  try {
    window.localStorage.setItem(
      RECIPIENT_STORAGE_KEY,
      JSON.stringify(
        recipientOverrides.value,
      ),
    )
  } catch {
    errorMessage.value =
      'Data penerima belum dapat disimpan.'
    return
  }

  recipientEditOrderID.value = ''

  successMessage.value =
    `Data penerima ${name} berhasil diperbarui.`
}

const paymentChannelLabels = {
  UNSPECIFIED: 'Belum ditentukan',
  CASH: 'Tunai',
  BANK_TRANSFER: 'Transfer Bank',
  VIRTUAL_ACCOUNT: 'Virtual Account',
  QRIS: 'QRIS',
  E_WALLET: 'Dompet Digital',
  CARD: 'Kartu Debit / Kredit',
  OTHER: 'Kanal Lainnya',
}

const paymentSettlementLabel = (
  value,
  channel = '',
) => {
  const settlement =
    String(
      value || '',
    ).toUpperCase()

  const normalizedChannel =
    String(
      channel || '',
    ).toUpperCase()

  if (settlement === 'TERM') {
    return 'KREDIT / TEMPO'
  }

  if (normalizedChannel === 'CASH') {
    return 'TUNAI'
  }

  return 'LUNAS LANGSUNG'
}

const paymentChannelLabel = (value) => {
  return (
    paymentChannelLabels[
      String(
        value || 'UNSPECIFIED',
      ).toUpperCase()
    ] ||
    'Belum ditentukan'
  )
}

const allDistributionOrders =
  computed(() => {
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
    )
  })

const getDistributionHistoryDate = order => {
  const status =
    String(
      order?.status ||
      '',
    ).toUpperCase()

  let value =
    null

  if (
    status ===
    'COMPLETED'
  ) {
    value =
      order?.completed_at ||
      order?.shipped_at
  }

  if (
    status ===
    'SHIPPED'
  ) {
    value =
      order?.shipped_at
  }

  if (!value) {
    return null
  }

  const date =
    new Date(value)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return null
  }

  return date
}

const isDistributionOrderVisible = order => {
  const status =
    String(
      order?.status ||
      '',
    ).toUpperCase()

  if (
    status ===
    'PACKED'
  ) {
    return true
  }

  if (
    ![
      'SHIPPED',
      'COMPLETED',
    ].includes(status)
  ) {
    return false
  }

  const historyDate =
    getDistributionHistoryDate(
      order,
    )

  if (!historyDate) {
    return true
  }

  const age =
    Date.now() -
    historyDate.getTime()

  return (
    age <=
    DISTRIBUTION_HISTORY_MS
  )
}

const distributionOrders =
  computed(() => {
    return allDistributionOrders.value.filter(
      isDistributionOrderVisible,
    )
  })

const archivedDistributionCount =
  computed(() => {
    return allDistributionOrders.value.filter(
      order =>
        !isDistributionOrderVisible(
          order,
        ),
    ).length
  })

const filteredOrders = computed(() => {
  const keyword =
    search.value
      .trim()
      .toLowerCase()

  return distributionOrders.value.filter(
    (order) => {
      const status =
        String(
          order.status || '',
        ).toUpperCase()

      const priority =
        String(
          order.priority || '',
        ).toUpperCase()

      if (
        statusFilter.value !== 'ALL' &&
        status !== statusFilter.value
      ) {
        return false
      }

      if (
        priorityFilter.value !== 'ALL' &&
        priority !== priorityFilter.value
      ) {
        return false
      }

      if (!keyword) {
        return true
      }

      const source = [
        order.order_number,
        order.customer_code,
        order.customer_name,
        order.cluster,
        order.priority,
        order.recommendation,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()

      return source.includes(keyword)
    },
  )
})

const packedCount = computed(() => {
  return distributionOrders.value.filter(
    (order) =>
      String(
        order.status || '',
      ).toUpperCase() === 'PACKED',
  ).length
})

const shippedCount = computed(() => {
  return distributionOrders.value.filter(
    (order) =>
      [
        'SHIPPED',
        'COMPLETED',
      ].includes(
        String(
          order.status || '',
        ).toUpperCase(),
      ),
  ).length
})

const priorityCount = computed(() => {
  return distributionOrders.value.filter(
    (order) =>
      [
        'VVIP',
        'HIGH_VOLUME',
      ].includes(
        String(
          order.priority || '',
        ).toUpperCase(),
      ),
  ).length
})

const readyValue = computed(() => {
  return distributionOrders.value
    .filter(
      (order) =>
        String(
          order.status || '',
        ).toUpperCase() === 'PACKED',
    )
    .reduce(
      (total, order) =>
        total +
        Number(
          order.total || 0,
        ),
      0,
    )
})

const formatCurrency = (value) => {
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

const formatNumber = (value) => {
  return new Intl.NumberFormat(
    'id-ID',
  ).format(
    Number(value || 0),
  )
}

const formatDate = (value) => {
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
    return value
  }

  return new Intl.DateTimeFormat(
    'id-ID',
    {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    },
  ).format(date)
}

const getDistributionHistoryLabel = order => {
  const status =
    String(
      order?.status ||
      '',
    ).toUpperCase()

  if (
    ![
      'SHIPPED',
      'COMPLETED',
    ].includes(status)
  ) {
    return ''
  }

  const historyDate =
    getDistributionHistoryDate(
      order,
    )

  if (!historyDate) {
    return (
      `Riwayat aktif maksimal ` +
      `${DISTRIBUTION_HISTORY_DAYS} hari`
    )
  }

  const expiresAt =
    new Date(
      historyDate.getTime() +
      DISTRIBUTION_HISTORY_MS,
    )

  return (
    `Tampil sampai ` +
    formatDate(
      expiresAt,
    ) +
    ` · arsip permanen tetap di Finance`
  )
}

const getPriorityLabel = (priority) => {
  const value =
    String(
      priority || '',
    ).toUpperCase()

  switch (value) {
  case 'HIGH_VOLUME':
    return 'HIGH VOLUME'

  case 'VVIP':
    return 'VVIP'

  case 'RETAIL':
    return 'RETAIL'

  default:
    return 'REGULAR'
  }
}

const getPriorityClass = (priority) => {
  const value =
    String(
      priority || '',
    ).toUpperCase()

  switch (value) {
  case 'HIGH_VOLUME':
    return 'priority-high-volume'

  case 'VVIP':
    return 'priority-vvip'

  case 'RETAIL':
    return 'priority-retail'

  default:
    return 'priority-regular'
  }
}

const formatInvoiceDate = (value) => {
  if (!value) {
    return '—'
  }

  const date = new Date(value)

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return '—'
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

const formatExpiryDate = (value) => {
  if (!value) {
    return '-'
  }

  return formatInvoiceDate(value)
}

const getPaymentTermsLabel = (days) => {
  const value = Number(days || 0)

  if (value <= 0) {
    return 'TUNAI'
  }

  return `${value} HARI`
}

const activeWarehouses = computed(() => {
  return warehouses.value.filter(
    (warehouse) =>
      warehouse.is_active,
  )
})

const loadWarehouses = async () => {
  warehouseLoading.value = true

  try {
    warehouses.value =
      await getWarehouses()
  } catch {
    warehouses.value = []
  } finally {
    warehouseLoading.value = false
  }
}

const activeTaxProfiles = computed(() => {
  return taxProfiles.value.filter(
    (profile) =>
      profile.is_active,
  )
})

const loadTaxProfiles = async () => {
  taxLoading.value = true

  try {
    const [
      profiles,
      current,
    ] = await Promise.all([
      getTaxProfiles(),
      getCurrentTaxProfile(),
    ])

    taxProfiles.value =
      Array.isArray(profiles)
        ? profiles
        : []

    currentTaxProfile.value =
      current || null
  } catch {
    taxProfiles.value = []
    currentTaxProfile.value = null
  } finally {
    taxLoading.value = false
  }
}

const cloneInvoiceForm = (invoice) => {
  return {
    sales_name:
      invoice?.sales_name || '',
    warehouse_id:
      invoice?.warehouse_id || '',
    payment_terms_days:
      Number(
        invoice?.payment_terms_days ||
        0,
      ),
    payment_settlement:
      invoice?.payment_settlement ||
      (
        Number(
          invoice?.payment_terms_days ||
          0,
        ) > 0
          ? 'TERM'
          : 'IMMEDIATE'
      ),
    payment_channel:
      invoice?.payment_channel &&
      invoice?.payment_channel !==
      'UNSPECIFIED'
        ? invoice.payment_channel
        : (
            Number(
              invoice?.payment_terms_days ||
              0,
            ) > 0
              ? 'BANK_TRANSFER'
              : 'CASH'
          ),
    payment_provider:
      invoice?.payment_provider || '',
    tax_mode:
      invoice?.tax_mode ||
      'STANDARD_PPN',
    tax_profile_id:
      invoice?.tax_profile_id ||
      currentTaxProfile.value?.id ||
      '',
    notes:
      invoice?.notes || '',
    items:
      (invoice?.items || []).map(
        (item) => ({
          id: item.id,
          batch_number:
            item.batch_number || '',
          expiry_date:
            item.expiry_date
              ? String(
                  item.expiry_date,
                ).slice(0, 10)
              : '',
          discount_percent:
            Number(
              item.discount_percent ||
              0,
            ),
        }),
      ),
  }
}

const invoiceFormDueDate = computed(() => {
  if (
    !invoiceForm.value ||
    invoiceForm.value.payment_settlement !==
      'TERM'
  ) {
    return ''
  }

  const days =
    Number(
      invoiceForm.value.payment_terms_days ||
      0,
    )

  if (
    !Number.isFinite(days) ||
    days <= 0 ||
    !invoiceData.value?.invoice_date
  ) {
    return ''
  }

  const date =
    new Date(
      invoiceData.value.invoice_date,
    )

  if (
    Number.isNaN(
      date.getTime(),
    )
  ) {
    return ''
  }

  date.setDate(
    date.getDate() +
      days,
  )

  return date.toISOString()
})

const buildInvoiceQRPayload = (invoice) => {
  return [
    `INV:${invoice?.invoice_number || ''}`,
    `SO:${invoice?.order_number || ''}`,
    `CUS:${invoice?.customer_code || ''}`,
    `TOTAL:${Math.round(
      Number(
        invoice?.total || 0,
      ),
    )}`,
  ].join('|')
}

const refreshInvoiceQR = async () => {
  if (!invoiceData.value) {
    invoiceQRCode.value = ''
    return
  }

  try {
    invoiceQRCode.value =
      await QRCode.toDataURL(
        buildInvoiceQRPayload(
          invoiceData.value,
        ),
        {
          width: 150,
          margin: 1,
          errorCorrectionLevel: 'M',
        },
      )
  } catch {
    invoiceQRCode.value = ''
  }
}

const openInvoice = async (order) => {
  if (
    !order?.id ||
    invoiceLoadingOrderID.value
  ) {
    return
  }

  errorMessage.value = ''
  invoiceLoadingOrderID.value =
    order.id

  try {
    if (
      warehouses.value.length === 0
    ) {
      await loadWarehouses()
    }

    if (
      taxProfiles.value.length === 0
    ) {
      await loadTaxProfiles()
    }

    activeInvoiceOrderID.value =
      order.id

    invoiceData.value =
      await ensureInvoiceBySalesOrder(
        order.id,
      )

    invoiceForm.value =
      cloneInvoiceForm(
        invoiceData.value,
      )


    invoiceEditing.value = false

    await refreshInvoiceQR()

    invoiceOpen.value = true
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Faktur tidak dapat dibuka. Silakan coba kembali.'
  } finally {
    invoiceLoadingOrderID.value = ''
  }
}

const closeInvoice = () => {
  invoiceOpen.value = false
  invoiceEditing.value = false
}

const startInvoiceEdit = () => {
  invoiceForm.value =
    cloneInvoiceForm(
      invoiceData.value,
    )

  invoiceEditing.value = true
}

const cancelInvoiceEdit = () => {
  invoiceForm.value =
    cloneInvoiceForm(
      invoiceData.value,
    )

  invoiceEditing.value = false
}

const taxRuleEffectivePreview = computed(() => {
  const nominal =
    Number(
      taxRuleForm.value.nominal_rate ||
      0,
    )

  const factor =
    Number(
      taxRuleForm.value.dpp_factor_percent ||
      0,
    ) / 100

  return (
    nominal *
    factor
  )
})

const openTaxRuleEditor = () => {
  const source =
    currentTaxProfile.value ||
    activeTaxProfiles.value[0] ||
    null

  taxRuleForm.value = {
    name:
      source?.name ||
      'PPN Standar',
    nominal_rate:
      Number(
        source?.nominal_rate ||
        12,
      ),
    dpp_factor_percent:
      Number(
        (
          Number(
            source?.dpp_factor ||
            0.91666667,
          ) * 100
        ).toFixed(4),
      ),
    regulation_reference:
      source?.regulation_reference ||
      '',
    notes:
      source?.notes ||
      '',
  }

  taxRuleEditorOpen.value = true
}

const closeTaxRuleEditor = () => {
  taxRuleEditorOpen.value = false
}

const saveTaxRule = async () => {
  if (!canManageTax.value) {
    return
  }

  taxRuleSaving.value = true
  errorMessage.value = ''

  try {
    const profile =
      await saveStandardTaxProfile({
        name:
          String(
            taxRuleForm.value.name ||
            'PPN Standar',
          ).trim(),
        nominal_rate:
          Number(
            taxRuleForm.value.nominal_rate ||
            0,
          ),
        dpp_factor_percent:
          Number(
            taxRuleForm.value.dpp_factor_percent ||
            0,
          ),
        regulation_reference:
          String(
            taxRuleForm.value.regulation_reference ||
            '',
          ).trim(),
        notes:
          String(
            taxRuleForm.value.notes ||
            '',
          ).trim(),
      })

    await loadTaxProfiles()

    if (invoiceForm.value) {
      invoiceForm.value.tax_mode =
        'STANDARD_PPN'
      invoiceForm.value.tax_profile_id =
        profile.id
    }

    taxRuleEditorOpen.value = false

    successMessage.value =
      `Aturan PPN ${profile.effective_year} berhasil disimpan dan menjadi aturan aktif.`
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Aturan PPN tidak dapat disimpan. Silakan coba kembali.'
  } finally {
    taxRuleSaving.value = false
  }
}

const saveInvoiceEdit = async () => {
  if (
    !invoiceData.value?.id ||
    !invoiceForm.value
  ) {
    return
  }

  invoiceSaving.value = true
  errorMessage.value = ''

  try {
    invoiceData.value =
      await updateInvoice(
        invoiceData.value.id,
        invoiceForm.value,
      )

    invoiceForm.value =
      cloneInvoiceForm(
        invoiceData.value,
      )

    invoiceEditing.value = false

    await refreshInvoiceQR()

    successMessage.value =
      invoiceData.value.is_printed
        ? `Faktur ${invoiceData.value.invoice_number} berhasil direvisi. Versi cetak diperbarui.`
        : `Faktur ${invoiceData.value.invoice_number} berhasil diperbarui.`
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Perubahan faktur tidak dapat disimpan. Silakan coba kembali.'
  } finally {
    invoiceSaving.value = false
  }
}

const printInvoice = async () => {
  if (
    !invoiceData.value?.id ||
    invoicePrinting.value
  ) {
    return
  }

  invoicePrinting.value = true

  const printModeClass =
    'fas-invoice-print-mode'

  const paperClass =
    printPaperMode.value === 'A4'
      ? 'fas-invoice-paper-a4'
      : 'fas-invoice-paper-sales'

  const cleanupPrintMode = () => {
    document.body.classList.remove(
      printModeClass,
    )

    document.body.classList.remove(
      'fas-invoice-paper-sales',
      'fas-invoice-paper-a4',
    )

    removeDynamicPrintPageStyle()
  }

  try {
    invoiceData.value =
      await markInvoicePrinted(
        invoiceData.value.id,
      )

    invoiceForm.value =
      cloneInvoiceForm(
        invoiceData.value,
      )

    await refreshInvoiceQR()
    await nextTick()

    installDynamicPrintPageStyle()

    document.body.classList.add(
      printModeClass,
      paperClass,
    )

    window.addEventListener(
      'afterprint',
      cleanupPrintMode,
      {
        once: true,
      },
    )

    await new Promise((resolve) => {
      requestAnimationFrame(() => {
        requestAnimationFrame(
          resolve,
        )
      })
    })

    window.print()
  } catch (error) {
    cleanupPrintMode()

    errorMessage.value =
      error.response?.data?.error ||
      'Faktur belum dapat dicetak. Silakan coba kembali.'
  } finally {
    invoicePrinting.value = false
  }
}

const terbilangInteger = (value) => {
  const angka = [
    '',
    'Satu',
    'Dua',
    'Tiga',
    'Empat',
    'Lima',
    'Enam',
    'Tujuh',
    'Delapan',
    'Sembilan',
    'Sepuluh',
    'Sebelas',
  ]

  const n = Math.floor(
    Number(value || 0),
  )

  if (n < 12) {
    return angka[n]
  }

  if (n < 20) {
    return `${terbilangInteger(
      n - 10,
    )} Belas`
  }

  if (n < 100) {
    return `${terbilangInteger(
      Math.floor(n / 10),
    )} Puluh ${terbilangInteger(
      n % 10,
    )}`.trim()
  }

  if (n < 200) {
    return `Seratus ${terbilangInteger(
      n - 100,
    )}`.trim()
  }

  if (n < 1000) {
    return `${terbilangInteger(
      Math.floor(n / 100),
    )} Ratus ${terbilangInteger(
      n % 100,
    )}`.trim()
  }

  if (n < 2000) {
    return `Seribu ${terbilangInteger(
      n - 1000,
    )}`.trim()
  }

  if (n < 1000000) {
    return `${terbilangInteger(
      Math.floor(n / 1000),
    )} Ribu ${terbilangInteger(
      n % 1000,
    )}`.trim()
  }

  if (n < 1000000000) {
    return `${terbilangInteger(
      Math.floor(n / 1000000),
    )} Juta ${terbilangInteger(
      n % 1000000,
    )}`.trim()
  }

  if (n < 1000000000000) {
    return `${terbilangInteger(
      Math.floor(n / 1000000000),
    )} Miliar ${terbilangInteger(
      n % 1000000000,
    )}`.trim()
  }

  return `${terbilangInteger(
    Math.floor(n / 1000000000000),
  )} Triliun ${terbilangInteger(
    n % 1000000000000,
  )}`.trim()
}

const invoiceTerbilang = computed(() => {
  if (!invoiceData.value) {
    return ''
  }

  return `${terbilangInteger(
    Math.round(
      Number(
        invoiceData.value.total ||
        0,
      ),
    ),
  )} Rupiah`
})

const getStatusLabel = (status) => {
  switch (
    String(
      status || '',
    ).toUpperCase()
  ) {
  case 'PACKED':
    return 'SIAP DIKIRIM'
  case 'SHIPPED':
    return 'DIKIRIM'
  case 'COMPLETED':
    return 'SELESAI'
  default:
    return String(
      status || '—',
    ).toUpperCase()
  }
}

const getStatusClass = (status) => {
  const value =
    String(
      status || '',
    ).toUpperCase()

  if (value === 'PACKED') {
    return 'status-packed'
  }

  if (
    value === 'SHIPPED' ||
    value === 'COMPLETED'
  ) {
    return 'status-shipped'
  }

  return 'status-neutral'
}

const toggleDetails = async (orderID) => {
  if (
    expandedOrderID.value ===
    orderID
  ) {
    expandedOrderID.value = ''
    return
  }

  expandedOrderID.value =
    orderID

  const current =
    orders.value.find(
      (order) =>
        order.id === orderID,
    )

  if (
    current?.items?.length
  ) {
    return
  }

  loadingDetailID.value =
    orderID

  try {
    const detail =
      await getSalesOrderByID(
        orderID,
      )

    const index =
      orders.value.findIndex(
        (order) =>
          order.id ===
          orderID,
      )

    if (index >= 0) {
      orders.value[index] = {
        ...orders.value[index],
        ...detail,
      }
    }
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Detail pesanan tidak dapat dimuat. Silakan coba kembali.'
  } finally {
    loadingDetailID.value = ''
  }
}

const markAsShipped = async (order) => {
  if (
    !order?.id ||
    shippingOrderID.value
  ) {
    return
  }

  errorMessage.value = ''
  successMessage.value = ''
  shippingOrderID.value =
    order.id

  try {
    const result =
      await shipSalesOrder(
        order.id,
      )

    const index =
      orders.value.findIndex(
        (item) =>
          item.id === order.id,
      )

    if (index >= 0) {
      orders.value[index] = {
        ...orders.value[index],
        ...result,
      }
    }

    successMessage.value =
      `${result.order_number} berhasil ditandai sebagai dikirim. Stok barang telah diperbarui.`
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Pengiriman belum dapat diselesaikan. Periksa ketersediaan stok dan coba kembali.'
  } finally {
    shippingOrderID.value = ''
  }
}

const loadOrders = async () => {
  loading.value = true
  errorMessage.value = ''

  try {
    orders.value =
      await getSalesOrders()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Data pengiriman tidak dapat dimuat. Silakan coba kembali.'
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await Promise.all([
    loadOrders(),
    loadWarehouses(),
    loadTaxProfiles(),
  ])
})
</script>

<template>
  <div class="distribution-page">
    <header class="page-header">
      <div>
        <p class="eyebrow">
          Manajemen Distribusi
        </p>

        <h1 class="page-title">
          Distribution
        </h1>

        <p class="page-description">
          Kelola pesanan siap kirim dan prioritas distribusi.
          Riwayat SHIPPED / COMPLETED tetap tampil
          {{ DISTRIBUTION_HISTORY_DAYS }} hari lalu otomatis
          keluar dari daftar Distribution tanpa menghapus data
          transaksi dari database.
        </p>
      </div>

      <button
        type="button"
        class="refresh-button"
        :disabled="loading"
        @click="loadOrders"
      >
        <span
          :class="[
            'refresh-icon',
            loading
              ? 'refresh-spinning'
              : '',
          ]"
        >
          <FasIcon name="refresh" />
        </span>

        {{
          loading
            ? 'Memuat...'
            : 'Perbarui'
        }}
      </button>
    </header>

    <div
      v-if="errorMessage"
      class="message error-message"
    >
      {{ errorMessage }}
    </div>

    <div
      v-if="successMessage"
      class="message success-message"
    >
      {{ successMessage }}
    </div>

    <section class="summary-grid">
      <article class="summary-card">
        <p class="summary-label">
          Siap Dikirim
        </p>

        <p class="summary-value">
          {{ packedCount }}
        </p>

        <p class="summary-helper">
          Pesanan selesai dikemas
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Sudah Dikirim
        </p>

        <p class="summary-value">
          {{ shippedCount }}
        </p>

        <p class="summary-helper">
          Riwayat aktif
          {{ DISTRIBUTION_HISTORY_DAYS }}
          hari
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Prioritas Tinggi
        </p>

        <p class="summary-value summary-priority">
          {{ priorityCount }}
        </p>

        <p class="summary-helper">
          VVIP + HIGH VOLUME
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Nilai Siap Kirim
        </p>

        <p class="summary-value summary-money">
          {{ formatCurrency(readyValue) }}
        </p>

        <p class="summary-helper">
          Nilai pesanan siap kirim
        </p>
      </article>
    </section>

    <div class="distribution-archive-info">
      <div>
        <strong>
          Arsip Otomatis Distribution
        </strong>

        <span>
          Pengiriman selesai hanya tampil
          {{ DISTRIBUTION_HISTORY_DAYS }} hari
          supaya daftar kerja gudang tidak menumpuk.
        </span>
      </div>

      <div class="distribution-archive-count">
        <strong>
          {{ archivedDistributionCount }}
        </strong>

        <span>
          transaksi sudah keluar dari daftar aktif
        </span>
      </div>

      <div class="distribution-archive-note">
        Tidak ada penghapusan PostgreSQL.
        Sales Order, customer, alamat, NPWP/NIK,
        faktur dan nilai penjualan tetap tersimpan
        dan dapat dicari di Finance.
      </div>
    </div>

    <section class="distribution-panel">
      <div class="toolbar">
        <div class="search-shell">
          <span class="search-icon">
            <FasIcon name="search" />
          </span>

          <input
            v-model="search"
            type="text"
            class="toolbar-input"
            placeholder="Cari nomor SO, pelanggan, kode, atau prioritas..."
          />
        </div>

        <FasSelect
          v-model="statusFilter"
          class="toolbar-select"
        >
          <option value="ALL">
            Semua Status
          </option>

          <option value="PACKED">
            Siap Dikirim
          </option>

          <option value="SHIPPED">
            Dikirim
          </option>

          <option value="COMPLETED">
            Completed
          </option>
        </FasSelect>

        <FasSelect
          v-model="priorityFilter"
          class="toolbar-select"
        >
          <option value="ALL">
            Semua Prioritas
          </option>

          <option value="VVIP">
            VVIP
          </option>

          <option value="HIGH_VOLUME">
            High Volume
          </option>

          <option value="REGULAR">
            Regular
          </option>

          <option value="RETAIL">
            Retail
          </option>
        </FasSelect>
      </div>

      <div
        v-if="loading"
        class="empty-state"
      >
        Memuat data distribusi...
      </div>

      <div
        v-else-if="
          distributionOrders.length === 0
        "
        class="empty-state empty-large"
      >
        <div class="empty-icon">
          ⇢
        </div>

        <h2>
          Belum ada pesanan siap dikirim
        </h2>

        <p>
          Pesanan akan muncul di sini setelah proses pengemasan selesai.
        </p>
      </div>

      <div
        v-else-if="
          filteredOrders.length === 0
        "
        class="empty-state"
      >
        Tidak ada pesanan yang sesuai dengan filter.
      </div>

      <div
        v-else
        class="shipment-list"
      >
        <article
          v-for="order in filteredOrders"
          :key="order.id"
          class="shipment-card"
        >
          <div class="shipment-main">
            <div class="shipment-identity">
              <div class="shipment-title-row">
                <h2>
                  {{ order.order_number }}
                </h2>

                <span
                  :class="[
                    'status-badge',
                    getStatusClass(
                      order.status,
                    ),
                  ]"
                >
                  {{
                    getStatusLabel(
                      order.status,
                    )
                  }}
                </span>
              </div>

              <p class="customer-name">
                {{ order.customer_name }}
              </p>

              <p class="customer-code">
                {{
                  order.customer_code
                }}
                ·
                {{
                  formatDate(
                    order.order_date,
                  )
                }}
              </p>

              <p
                v-if="
                  [
                    'SHIPPED',
                    'COMPLETED',
                  ].includes(
                    String(
                      order.status ||
                      '',
                    ).toUpperCase(),
                  )
                "
                class="shipment-history-retention"
              >
                {{
                  getDistributionHistoryLabel(
                    order,
                  )
                }}
              </p>
            </div>

            <div class="shipment-metric">
              <span>
                Qty
              </span>

              <strong>
                {{
                  formatNumber(
                    order.cl_qty,
                  )
                }}
              </strong>
            </div>

            <div class="shipment-metric">
              <span>
                Total
              </span>

              <strong>
                {{
                  formatCurrency(
                    order.total,
                  )
                }}
              </strong>
            </div>

            <div class="shipment-priority">
              <span>
                Prioritas
              </span>

              <strong
                :class="
                  getPriorityClass(
                    order.priority,
                  )
                "
              >
                {{
                  getPriorityLabel(
                    order.priority,
                  )
                }}
              </strong>

              <small>
                {{
                  order.cluster ||
                  'Belum ada cluster'
                }}
              </small>
            </div>

            <div class="shipment-actions">
              <button
                type="button"
                class="detail-button"
                @click="
                  toggleDetails(
                    order.id,
                  )
                "
              >
                {{
                  expandedOrderID ===
                  order.id
                    ? 'Tutup Detail'
                    : 'Lihat Detail'
                }}
              </button>

              <button
                v-if="
                  String(
                    order.status,
                  ).toUpperCase() ===
                  'PACKED'
                "
                type="button"
                class="ship-button"
                :disabled="
                  !canManageDistribution ||
                  shippingOrderID ===
                    order.id
                "
                @click="
                  markAsShipped(
                    order,
                  )
                "
              >
                {{
                  shippingOrderID ===
                  order.id
                    ? 'Memproses...'
                    : 'Tandai Dikirim'
                }}
              </button>

              <button
                v-if="
                  [
                    'PACKED',
                    'SHIPPED',
                    'COMPLETED',
                  ].includes(
                    String(
                      order.status,
                    ).toUpperCase(),
                  )
                "
                type="button"
                class="invoice-button"
                :disabled="
                  invoiceLoadingOrderID ===
                    order.id
                "
                @click="
                  openInvoice(
                    order,
                  )
                "
              >
                {{
                  invoiceLoadingOrderID ===
                  order.id
                    ? 'Membuka...'
                    : 'Lihat Faktur'
                }}
              </button>
            </div>
          </div>

          <div
            v-if="
              expandedOrderID ===
              order.id
            "
            class="shipment-detail"
          >
            <div
              v-if="
                loadingDetailID ===
                order.id
              "
              class="detail-loading"
            >
              Memuat detail pesanan...
            </div>

            <div
              v-else
              class="detail-grid"
            >
              <div class="detail-box detail-customer-box">
                <div class="detail-box-heading">
                  <span>
                    Penerima Faktur
                  </span>

                  <button
                    v-if="
                      canManageDistribution &&
                      recipientEditOrderID !==
                        order.id
                    "
                    type="button"
                    class="detail-edit-button"
                    @click="
                      beginRecipientEdit(
                        order,
                      )
                    "
                  >
                    Edit
                  </button>
                </div>

                <template
                  v-if="
                    recipientEditOrderID !==
                    order.id
                  "
                >
                  <strong>
                    {{
                      getOrderRecipientName(
                        order,
                      )
                    }}
                  </strong>

                  <small>
                    {{
                      order.customer_code
                    }}
                  </small>

                  <small
                    v-if="
                      getOrderRecipientAddress(
                        order,
                      )
                    "
                    class="detail-address"
                  >
                    {{
                      getOrderRecipientAddress(
                        order,
                      )
                    }}
                  </small>
                </template>

                <div
                  v-else
                  class="recipient-edit-form"
                >
                  <label>
                    <span>Nama Penerima</span>

                    <input
                      v-model="
                        recipientForm.name
                      "
                      type="text"
                      placeholder="Nama toko / penerima"
                    />
                  </label>

                  <label>
                    <span>Alamat Tujuan</span>

                    <textarea
                      v-model="
                        recipientForm.address
                      "
                      rows="3"
                      placeholder="Alamat yang tampil pada faktur"
                    ></textarea>
                  </label>

                  <div class="recipient-edit-actions">
                    <button
                      type="button"
                      class="recipient-cancel-button"
                      @click="
                        cancelRecipientEdit
                      "
                    >
                      Batal
                    </button>

                    <button
                      type="button"
                      class="recipient-save-button"
                      @click="
                        saveRecipientEdit(
                          order,
                        )
                      "
                    >
                      Simpan
                    </button>
                  </div>
                </div>
              </div>

              <div class="detail-box">
                <span>
                  Klasifikasi Prioritas
                </span>

                <strong>
                  {{
                    order.cluster ||
                    '—'
                  }}
                </strong>

                <small>
                  {{
                    getPriorityLabel(
                      order.priority,
                    )
                  }}
                </small>
              </div>

              <div class="detail-box">
                <span>
                  Rekomendasi
                </span>

                <strong class="detail-long">
                  {{
                    order.recommendation ||
                    '—'
                  }}
                </strong>
              </div>

              <div class="detail-box">
                <span>
                  Catatan
                </span>

                <strong class="detail-long">
                  {{
                    order.notes ||
                    'Tidak ada catatan'
                  }}
                </strong>
              </div>
            </div>

            <div
              v-if="
                loadingDetailID !==
                  order.id &&
                order.items &&
                order.items.length
              "
              class="detail-table-scroll"
            >
              <table class="detail-table">
                <thead>
                  <tr>
                    <th>Produk</th>
                    <th>Qty</th>
                    <th>Unit</th>
                    <th>Harga</th>
                    <th>Subtotal</th>
                  </tr>
                </thead>

                <tbody>
                  <tr
                    v-for="item in order.items"
                    :key="item.id"
                  >
                    <td>
                      <strong>
                        {{
                          item.product_name
                        }}
                      </strong>

                      <small>
                        {{
                          item.product_code
                        }}
                        ·
                        {{
                          item.sku
                        }}
                      </small>
                    </td>

                    <td>
                      {{
                        item.quantity
                      }}
                    </td>

                    <td>
                      {{
                        item.unit
                      }}
                    </td>

                    <td>
                      {{
                        formatCurrency(
                          item.unit_price,
                        )
                      }}
                    </td>

                    <td>
                      {{
                        formatCurrency(
                          item.subtotal,
                        )
                      }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>

            <div class="distribution-note">
              {{
                String(
                  order.status,
                ).toUpperCase() ===
                'PACKED'
                  ? 'Pesanan siap dikirim. Pastikan jumlah barang dan tujuan pengiriman sudah sesuai sebelum melanjutkan.'
                  : 'Pengiriman telah dicatat dan stok barang sudah diperbarui.'
              }}
            </div>
          </div>
        </article>
      </div>
    </section>
  </div>

  <Teleport to="body">
    <div
      v-if="
        invoiceOpen &&
        invoiceData
      "
      class="invoice-print-overlay"
    >
      <div class="invoice-print-toolbar">
        <div class="invoice-toolbar-status">
          <strong>
            {{ invoiceData.invoice_number }}
          </strong>

          <span>
            {{
              invoiceData.is_printed
                ? `Sudah dicetak ${invoiceData.print_count}x · Versi ${invoiceData.print_version}`
                : 'Belum dicetak'
            }}
          </span>
        </div>

        <div class="invoice-paper-toolbar-control">
          <span class="invoice-paper-toolbar-label">
            Ukuran Cetak
          </span>

          <div class="invoice-paper-segmented">
            <button
              v-for="paper in printPaperOptions"
              :key="paper.value"
              type="button"
              :class="[
                'invoice-paper-option',
                printPaperMode === paper.value
                  ? 'invoice-paper-option-active'
                  : '',
              ]"
              @click="
                setPrintPaperMode(
                  paper.value,
                )
              "
            >
              <strong>
                {{ paper.label }}
              </strong>

              <small>
                {{ paper.shortLabel }}
              </small>
            </button>
          </div>

          <span class="invoice-paper-page-count">
            {{
              invoicePrintPageCount
            }}
            halaman
          </span>
        </div>

        <div class="invoice-toolbar-actions">
          <button
            type="button"
            class="invoice-toolbar-secondary"
            @click="closeInvoice"
          >
            Tutup
          </button>

          <button
            v-if="!invoiceEditing"
            type="button"
            class="invoice-toolbar-edit"
            @click="startInvoiceEdit"
          >
            ✎
            {{
              invoiceData.is_printed
                ? 'Edit / Revisi'
                : 'Edit Faktur'
            }}
          </button>

          <button
            v-if="invoiceEditing"
            type="button"
            class="invoice-toolbar-secondary"
            @click="cancelInvoiceEdit"
          >
            Batal
          </button>

          <button
            v-if="invoiceEditing"
            type="button"
            class="invoice-toolbar-save"
            :disabled="invoiceSaving"
            @click="saveInvoiceEdit"
          >
            {{
              invoiceSaving
                ? 'Menyimpan...'
                : 'Simpan Perubahan'
            }}
          </button>

          <button
            v-if="!invoiceEditing"
            type="button"
            class="invoice-toolbar-primary"
            :disabled="invoicePrinting"
            @click="printInvoice"
          >
            {{
              invoicePrinting
                ? 'Menyiapkan...'
                : (
                  invoiceData.is_printed
                    ? 'Cetak Ulang / PDF'
                    : 'Cetak / Simpan PDF'
                )
            }}
          </button>
        </div>
      </div>

      <section
        v-if="
          invoiceEditing &&
          invoiceForm
        "
        class="invoice-editor-panel"
      >
        <div class="invoice-editor-heading">
          <div>
            <span>Koreksi Faktur</span>
            <strong>Periksa informasi sebelum dicetak</strong>
          </div>

          <small>
            Perubahan setelah faktur dicetak akan tercatat sebagai revisi.
          </small>
        </div>

        <div class="invoice-editor-grid">
          <section class="invoice-company-address-editor">
            <div class="invoice-company-address-heading">
              <div>
                <span>Identitas Perusahaan</span>
                <strong>{{ invoiceIssuer.name }}</strong>
              </div>

              <small>
                Nama perusahaan, nomor izin, dan NPWP bersifat tetap.
              </small>
            </div>

            <div class="invoice-company-fixed-grid">
              <div class="invoice-company-fixed-item">
                <span>Nama Perusahaan</span>
                <strong>{{ invoiceIssuer.name }}</strong>
              </div>

              <div class="invoice-company-fixed-item">
                <span>No. Izin</span>
                <strong>{{ invoiceIssuer.license_number }}</strong>
              </div>

              <div class="invoice-company-fixed-item">
                <span>NPWP</span>
                <strong>{{ invoiceIssuer.tax_id }}</strong>
              </div>
            </div>

            <div class="invoice-edit-field invoice-company-address-field">
              <span>Alamat Perusahaan (profil pusat)</span>
              <p style="white-space:pre-line">{{ companyAddress }}</p>
              <small>Perubahan identitas hanya melalui Settings → Profil Perusahaan oleh owner. Faktur yang sudah memiliki snapshot tetap memakai identitas penerbitnya.</small>
            </div>
          </section>

          <label class="invoice-edit-field">
            <span>Salesman</span>

            <input
              v-model="invoiceForm.sales_name"
              type="text"
              placeholder="Contoh: AGUNG"
            />
          </label>

          <label class="invoice-edit-field">
            <span>Gudang Asal</span>

            <FasSelect
              v-model="invoiceForm.warehouse_id"
              :disabled="warehouseLoading"
            >
              <option value="" disabled>
                Pilih Gudang
              </option>

              <option
                v-for="warehouse in activeWarehouses"
                :key="warehouse.id"
                :value="warehouse.id"
              >
                {{
                  warehouse.name
                }}
                ·
                {{
                  warehouse.classification
                }}
              </option>
            </FasSelect>
          </label>

          <label class="invoice-edit-field">
            <span>Skema Pembayaran</span>

            <FasSelect
              v-model="invoiceForm.payment_settlement"
            >
              <option value="IMMEDIATE">
                Pembayaran Langsung
              </option>

              <option value="TERM">
                Pembayaran Tempo
              </option>
            </FasSelect>
          </label>

          <label class="invoice-edit-field">
            <span>Kanal Pembayaran</span>

            <FasSelect
              v-model="invoiceForm.payment_channel"
            >
              <option value="CASH">
                Tunai
              </option>

              <option value="BANK_TRANSFER">
                Transfer Bank
              </option>

              <option value="VIRTUAL_ACCOUNT">
                Virtual Account
              </option>

              <option value="QRIS">
                QRIS
              </option>

              <option value="E_WALLET">
                Dompet Digital
              </option>

              <option value="CARD">
                Kartu Debit / Kredit
              </option>

              <option value="OTHER">
                Kanal Lainnya
              </option>
            </FasSelect>
          </label>

          <label
            v-if="
              invoiceForm.payment_settlement ===
              'TERM'
            "
            class="invoice-edit-field"
          >
            <span>Tempo Pembayaran</span>

            <div class="invoice-edit-number">
              <input
                v-model.number="
                  invoiceForm.payment_terms_days
                "
                type="number"
                min="1"
                max="365"
              />

              <strong>HARI</strong>
            </div>
          </label>

          <div
            v-if="
              invoiceForm.payment_settlement ===
              'TERM'
            "
            class="invoice-tax-preview invoice-due-preview"
          >
            <span>Tanggal Jatuh Tempo</span>

            <strong>
              {{
                invoiceFormDueDate
                  ? formatInvoiceDate(
                      invoiceFormDueDate,
                    )
                  : '—'
              }}
            </strong>

            <small>
              Dihitung dari tanggal faktur
            </small>
          </div>

          <label
            v-if="
              [
                'BANK_TRANSFER',
                'VIRTUAL_ACCOUNT',
                'E_WALLET',
                'CARD',
                'OTHER',
              ].includes(
                invoiceForm.payment_channel,
              )
            "
            class="invoice-edit-field"
          >
            <span>Bank / Provider</span>

            <input
              v-model="
                invoiceForm.payment_provider
              "
              type="text"
              placeholder="Contoh: BCA, Mandiri, BRI, Bank Jago..."
            />
          </label>

          <label class="invoice-edit-field">
            <span>Perlakuan Pajak</span>

            <FasSelect
              v-model="invoiceForm.tax_mode"
            >
              <option value="STANDARD_PPN">
                PPN Standar
              </option>

              <option value="NON_PPN">
                Non-PPN
              </option>
            </FasSelect>
          </label>

          <label
            v-if="
              invoiceForm.tax_mode ===
              'STANDARD_PPN'
            "
            class="invoice-edit-field"
          >
            <span>Aturan PPN</span>

            <FasSelect
              v-model="
                invoiceForm.tax_profile_id
              "
              :disabled="taxLoading"
            >
              <option value="" disabled>
                Pilih aturan PPN
              </option>

              <option
                v-for="profile in activeTaxProfiles"
                :key="profile.id"
                :value="profile.id"
              >
                {{
                  profile.name
                }}
                · efektif
                {{
                  Number(
                    profile.effective_rate ||
                    0,
                  )
                }}%
                ·
                {{
                  profile.effective_year
                }}
              </option>
            </FasSelect>
          </label>
        </div>

        <div
          v-if="
            invoiceForm.tax_mode ===
            'STANDARD_PPN'
          "
          class="invoice-tax-rule-strip"
        >
          <div>
            <span>Aturan PPN aktif</span>

            <strong>
              {{
                activeTaxProfiles.find(
                  (profile) =>
                    profile.id ===
                    invoiceForm.tax_profile_id,
                )?.name ||
                currentTaxProfile?.name ||
                'PPN Standar'
              }}
            </strong>

            <small>
              Tarif
              {{
                Number(
                  (
                    activeTaxProfiles.find(
                      (profile) =>
                        profile.id ===
                        invoiceForm.tax_profile_id,
                    ) ||
                    currentTaxProfile
                  )?.nominal_rate ||
                  0,
                )
              }}%
              · DPP
              {{
                Number(
                  (
                    (
                      activeTaxProfiles.find(
                        (profile) =>
                          profile.id ===
                          invoiceForm.tax_profile_id,
                      ) ||
                      currentTaxProfile
                    )?.dpp_factor ||
                    0
                  ) * 100,
                ).toFixed(4)
              }}%
              · Efektif
              {{
                Number(
                  (
                    activeTaxProfiles.find(
                      (profile) =>
                        profile.id ===
                        invoiceForm.tax_profile_id,
                    ) ||
                    currentTaxProfile
                  )?.effective_rate ||
                  0,
                )
              }}%
            </small>
          </div>

          <button
            v-if="canManageTax"
            type="button"
            class="invoice-tax-rule-edit"
            @click="openTaxRuleEditor"
          >
            ✎ Ubah Aturan PPN
          </button>
        </div>

        <div
          v-if="
            taxRuleEditorOpen &&
            canManageTax
          "
          class="invoice-tax-rule-editor"
        >
          <div class="invoice-tax-rule-heading">
            <div>
              <span>Master Pajak</span>
              <strong>
                Aturan PPN Standar
              </strong>
            </div>

            <button
              type="button"
              @click="closeTaxRuleEditor"
            >
              <FasIcon name="close" />
            </button>
          </div>

          <div class="invoice-tax-rule-grid">
            <label class="invoice-edit-field">
              <span>Nama Aturan</span>

              <input
                v-model="
                  taxRuleForm.name
                "
                type="text"
                placeholder="PPN Standar"
              />
            </label>

            <label class="invoice-edit-field">
              <span>Tarif PPN (%)</span>

              <input
                v-model.number="
                  taxRuleForm.nominal_rate
                "
                type="number"
                min="0"
                max="100"
                step="0.01"
              />
            </label>

            <label class="invoice-edit-field">
              <span>Faktor DPP (%)</span>

              <input
                v-model.number="
                  taxRuleForm.dpp_factor_percent
                "
                type="number"
                min="0"
                max="200"
                step="0.0001"
              />
            </label>

            <div class="invoice-tax-preview">
              <span>Tarif Efektif</span>

              <strong>
                {{
                  Number(
                    taxRuleEffectivePreview,
                  ).toFixed(4)
                }}%
              </strong>

              <small>
                Tahun aktif ditetapkan otomatis saat disimpan
              </small>
            </div>

            <label class="invoice-edit-field invoice-tax-rule-reference">
              <span>Referensi Peraturan</span>

              <input
                v-model="
                  taxRuleForm.regulation_reference
                "
                type="text"
                placeholder="Contoh: PMK ... / Tahun ..."
              />
            </label>

            <label class="invoice-edit-field invoice-tax-rule-reference">
              <span>Catatan</span>

              <input
                v-model="
                  taxRuleForm.notes
                "
                type="text"
                placeholder="Opsional"
              />
            </label>
          </div>

          <div class="invoice-tax-rule-actions">
            <button
              type="button"
              class="invoice-toolbar-secondary"
              @click="closeTaxRuleEditor"
            >
              Batal
            </button>

            <button
              type="button"
              class="invoice-toolbar-save"
              :disabled="taxRuleSaving"
              @click="saveTaxRule"
            >
              {{
                taxRuleSaving
                  ? 'Menyimpan...'
                  : 'Simpan sebagai Aturan Aktif'
              }}
            </button>
          </div>
        </div>

        <label class="invoice-edit-field invoice-edit-notes">
          <span>Catatan Faktur</span>
          <textarea
            v-model="invoiceForm.notes"
            rows="2"
            placeholder="Catatan tambahan bila diperlukan"
          ></textarea>
        </label>

        <div class="invoice-editor-items">
          <div class="invoice-editor-items-header">
            <strong>Batch, ED, dan Diskon</strong>
            <span>
              Quantity dan harga mengikuti Sales Order yang sudah dikirim.
            </span>
          </div>

          <div
            v-for="(editItem, index) in invoiceForm.items"
            :key="editItem.id"
            class="invoice-editor-item-row"
          >
            <div class="invoice-editor-item-name">
              <strong>
                {{ invoiceData.items[index]?.product_name }}
              </strong>
              <span>
                {{ invoiceData.items[index]?.product_code }}
              </span>
            </div>

            <label>
              <span>No. Batch</span>
              <input
                v-model="editItem.batch_number"
                type="text"
                placeholder="Nomor batch"
              />
            </label>

            <label>
              <span>ED</span>
              <input
                v-model="editItem.expiry_date"
                type="date"
              />
            </label>

            <label>
              <span>Disc %</span>
              <input
                v-model.number="editItem.discount_percent"
                type="number"
                min="0"
                max="100"
                step="0.01"
              />
            </label>
          </div>
        </div>
      </section>

      <div
        :class="[
          'invoice-pages-preview',
          printPaperMode === 'A4'
            ? 'invoice-pages-a4'
            : 'invoice-pages-sales',
        ]"
      >
        <article
          v-for="page in invoicePrintPages"
          :key="`${invoiceData.id}-${page.number}`"
          :class="[
            'invoice-document',
            page.isLast
              ? 'invoice-document-last'
              : 'invoice-document-continuation',
          ]"
        >
          <header class="invoice-original-header">
            <div class="invoice-company-box">
              <strong>{{ invoiceIssuer.name }}</strong>

              <span
                v-for="(
                  line,
                  lineIndex
                ) in companyAddressLines"
                :key="
                  `company-address-${lineIndex}`
                "
              >
                {{ line }}
              </span>

              <span>
                No. Izin : {{ invoiceIssuer.license_number }}
              </span>

              <span>
                NPWP : {{ invoiceIssuer.tax_id }}
              </span>
            </div>

            <div class="invoice-middle-meta">
              <div>
                <span>No. SJ</span>
                <strong>
                  {{
                    invoiceData.delivery_note_number
                  }}
                </strong>
              </div>

              <div>
                <span>Tgl.</span>
                <strong>
                  {{
                    formatInvoiceDate(
                      invoiceData.invoice_date,
                    )
                  }}
                </strong>
              </div>

              <div>
                <span>Tgl J. Tempo</span>
                <strong>
                  {{
                    formatInvoiceDate(
                      invoiceData.due_date,
                    )
                  }}
                </strong>
              </div>

              <div>
                <span>Salesman</span>
                <strong>
                  {{
                    invoiceData.sales_name ||
                    '—'
                  }}
                </strong>
              </div>

              <div>
                <span>Skema Bayar</span>

                <strong>
                  {{
                    paymentSettlementLabel(
                      invoiceData.payment_settlement,
                      invoiceData.payment_channel,
                    )
                  }}
                </strong>
              </div>

              <div>
                <span>Kanal Bayar</span>

                <strong>
                  {{
                    paymentChannelLabel(
                      invoiceData.payment_channel,
                    )
                  }}
                </strong>

                <span
                  v-if="
                    invoiceData.payment_provider
                  "
                >
                  {{
                    invoiceData.payment_provider
                  }}
                </span>
              </div>

              <div>
                <span>Tempo</span>

                <strong>
                  {{
                    invoiceData.payment_settlement ===
                    'TERM'
                      ? getPaymentTermsLabel(
                          invoiceData.payment_terms_days,
                        )
                      : 'LANGSUNG'
                  }}
                </strong>
              </div>

              <div>
                <span>Gudang</span>

                <strong>
                  {{
                    invoiceData.warehouse_name ||
                    '—'
                  }}
                </strong>

                <span
                  v-if="
                    invoiceData.warehouse_classification
                  "
                >
                  {{
                    invoiceData.warehouse_classification
                  }}
                </span>
              </div>
            </div>

            <div class="invoice-customer-box">
              <span>Kepada Yth,</span>

              <strong>
                {{
                  invoiceRecipientName
                }}
              </strong>

              <span class="invoice-customer-address">
                {{
                  invoiceRecipientAddress
                }}
              </span>

              <span
                v-if="
                  invoiceData.customer_tax_id_number
                "
              >
                NPWP/NIK :
                {{
                  invoiceData.customer_tax_id_number
                }}
              </span>

              <span>
                {{ invoiceData.customer_code }}
              </span>
            </div>
          </header>

          <div class="invoice-reference-row">
            <div>
              <span>No. Faktur</span>
              <strong>
                {{
                  invoiceData.invoice_number
                }}
              </strong>
            </div>

            <div>
              <span>No. Sales Order</span>
              <strong>
                {{
                  invoiceData.order_number
                }}
              </strong>
            </div>

            <div>
              <span>
                Halaman
              </span>

              <strong>
                {{
                  page.number
                }}
                /
                {{
                  invoicePrintPageCount
                }}
              </strong>
            </div>
          </div>

          <div class="invoice-table-scroll">
            <table class="invoice-paper-table">
              <thead>
                <tr>
                  <th>NO.</th>
                  <th>NAMA BARANG</th>
                  <th>NO BATCH</th>
                  <th>ED</th>
                  <th>QUANTITY</th>
                  <th>HARGA</th>
                  <th>DISC %</th>
                  <th>DISC RP</th>
                  <th>JUMLAH</th>
                </tr>
              </thead>

              <tbody>
                <tr
                  v-for="item in page.items"
                  :key="item.id"
                >
                  <td>
                    {{
                      item.__originalIndex + 1
                    }}
                  </td>

                  <td class="invoice-paper-product">
                    <strong>
                      {{ item.product_name }}
                    </strong>

                    <small>
                      {{
                        item.product_code
                      }}
                      ·
                      {{
                        item.sku
                      }}
                    </small>
                  </td>

                  <td>
                    {{
                      item.batch_number ||
                      '-'
                    }}
                  </td>

                  <td>
                    {{
                      formatExpiryDate(
                        item.expiry_date,
                      )
                    }}
                  </td>

                  <td>
                    {{
                      item.qty
                    }}
                    {{
                      item.unit
                    }}
                  </td>

                  <td>
                    {{
                      formatCurrency(
                        item.price,
                      )
                    }}
                  </td>

                  <td>
                    {{
                      Number(
                        item.discount_percent ||
                        0,
                      )
                    }}
                  </td>

                  <td>
                    {{
                      formatCurrency(
                        item.discount_amount,
                      )
                    }}
                  </td>

                  <td>
                    {{
                      formatCurrency(
                        item.subtotal,
                      )
                    }}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div
            v-if="!page.isLast"
            class="invoice-continuation-footer"
          >
            <span>
              Bersambung ke halaman
              {{
                page.number + 1
              }}
            </span>

            <strong>
              {{
                invoiceData.invoice_number
              }}
            </strong>
          </div>

          <template v-if="page.isLast">
            <section class="invoice-bottom-grid">
              <div class="invoice-signature-box">
                <div>
                  <span>
                    Tanda terima,
                  </span>

                  <strong>
                    Nama Lengkap / Stempel
                  </strong>
                </div>

                <div>
                  <span>
                    Hormat kami,
                  </span>

                  <strong>
                    PT Furi Anugerah Sejahtera
                  </strong>
                </div>
              </div>

              <div class="invoice-tax-box">
                <div>
                  <span>Total</span>
                  <strong>
                    {{
                      formatCurrency(
                        invoiceData.subtotal,
                      )
                    }}
                  </strong>
                </div>

                <div>
                  <span>Discount</span>
                  <strong>
                    {{
                      formatCurrency(
                        invoiceData.discount_amount,
                      )
                    }}
                  </strong>
                </div>

                <div>
                  <span>
                    DPP
                    {{
                      invoiceData.tax_mode ===
                      'STANDARD_PPN'
                        ? 'Nilai Lain'
                        : ''
                    }}
                  </span>

                  <strong>
                    {{
                      formatCurrency(
                        invoiceData.tax_base_amount,
                      )
                    }}
                  </strong>
                </div>

                <div>
                  <span>
                    {{
                      invoiceData.tax_mode ===
                      'STANDARD_PPN'
                        ? `PPN ${Number(invoiceData.tax_rate || 0)}% · efektif ${Number(invoiceData.effective_tax_rate || 0)}%${invoiceData.tax_effective_year ? ` · ${invoiceData.tax_effective_year}` : ''}`
                        : 'PPN'
                    }}
                  </span>

                  <strong>
                    {{
                      formatCurrency(
                        invoiceData.tax_amount,
                      )
                    }}
                  </strong>
                </div>

                <div class="invoice-tax-total">
                  <span>Jumlah</span>

                  <strong>
                    {{
                      formatCurrency(
                        invoiceData.total,
                      )
                    }}
                  </strong>
                </div>
              </div>
            </section>

            <section class="invoice-paper-footer">
              <div class="invoice-terbilang">
                <strong>
                  Nilai dalam Huruf :
                </strong>

                <span>
                  {{ invoiceTerbilang }}
                </span>
              </div>

              <div class="invoice-qr-block">
                <img
                  v-if="invoiceQRCode"
                  :src="invoiceQRCode"
                  alt="QR Faktur"
                />

                <div>
                  <strong>
                    {{
                      invoiceData.invoice_number
                    }}
                  </strong>

                  <span>
                    QR Faktur
                  </span>
                </div>
              </div>
            </section>

            <div
              v-if="invoiceData.is_printed"
              class="invoice-version-note"
            >
              Versi
              {{
                invoiceData.print_version
              }}
              · Cetak
              {{
                invoiceData.print_count
              }}
            </div>
          </template>
        </article>
      </div>
    </div>
  </Teleport>

</template>

<style scoped>
.distribution-page {
width: 100%;
  max-width: 100%;
  min-width: 0;
  padding: 1.35rem;
  color: var(--fas-text);
  container-type: inline-size;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1.5rem;
}

.eyebrow {
  color: var(--fas-accent-text);
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.page-title {
  margin-top: 0.45rem;
  color: var(--fas-text);
  font-size: 2rem;
  font-weight: 850;
  letter-spacing: -0.03em;
}

.page-description {
  max-width: 48rem;
  margin-top: 0.65rem;
  color: var(--fas-muted);
  font-size: 0.8rem;
  line-height: 1.4rem;
}

.refresh-button {
  min-height: 2.85rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.8rem;
  background: var(--fas-card);
  padding: 0 0.9rem;
  color: var(--fas-text-soft);
  font-size: 0.68rem;
  font-weight: 800;
}

.refresh-button:hover:not(:disabled) {
  border-color: rgba(96, 165, 250, 0.28);
  background: rgba(37, 99, 235, 0.1);
  color: #dbeafe;
}

.refresh-button:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.refresh-icon {
  display: inline-block;
  font-size: 0.9rem;
}

.refresh-spinning {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.message {
  margin-top: 1rem;
  border-radius: 0.9rem;
  padding: 0.85rem 1rem;
  font-size: 0.7rem;
}

.error-message {
  border: 1px solid rgba(248, 113, 113, 0.22);
  background: rgba(248, 113, 113, 0.07);
  color: #f87171;
}

.success-message {
  border: 1px solid rgba(52, 211, 153, 0.22);
  background: rgba(16, 185, 129, 0.08);
  color: #34d399;
}

.summary-grid {
  margin-top: 1.4rem;
  display: grid;
  grid-template-columns:
    repeat(4, minmax(0, 1fr));
  gap: 0.9rem;
}

.summary-card,
.distribution-archive-info {
  display:
    grid;
  grid-template-columns:
    minmax(
      0,
      1fr
    )
    auto;
  gap:
    0.65rem
    1rem;
  margin-top:
    1rem;
  border:
    1px solid
    rgba(
      59,
      130,
      246,
      0.18
    );
  border-radius:
    0.82rem;
  background:
    rgba(
      37,
      99,
      235,
      0.06
    );
  padding:
    0.75rem
    0.85rem;
}

.distribution-archive-info
> div:first-child {
  display:
    grid;
  gap:
    0.2rem;
}

.distribution-archive-info
> div:first-child
strong {
  color:
    #93c5fd;
  font-size:
    0.62rem;
}

.distribution-archive-info
> div:first-child
span {
  color:
    var(--fas-muted);
  font-size:
    0.53rem;
  line-height:
    0.9rem;
}

.distribution-archive-count {
  display:
    flex;
  align-items:
    center;
  gap:
    0.48rem;
  border:
    1px solid
    rgba(
      59,
      130,
      246,
      0.15
    );
  border-radius:
    0.64rem;
  background:
    rgba(
      15,
      23,
      42,
      0.45
    );
  padding:
    0.5rem
    0.65rem;
}

.distribution-archive-count
strong {
  color:
    #60a5fa;
  font-size:
    0.9rem;
}

.distribution-archive-count
span {
  max-width:
    10rem;
  color:
    #64748b;
  font-size:
    0.46rem;
  line-height:
    0.7rem;
}

.distribution-archive-note {
  grid-column:
    1 / -1;
  border-top:
    1px solid
    rgba(
      59,
      130,
      246,
      0.1
    );
  padding-top:
    0.52rem;
  color:
    #64748b;
  font-size:
    0.5rem;
  line-height:
    0.82rem;
}

.shipment-history-retention {
  max-width:
    30rem;
  margin-top:
    0.26rem;
  color:
    #64748b;
  font-size:
    0.46rem;
  line-height:
    0.72rem;
}

.distribution-panel {
  border: 1px solid var(--fas-border);
  background: var(--fas-card);
  box-shadow: 0 18px 45px var(--fas-shadow);
}

.summary-card {
  border-radius: 1.1rem;
  padding: 1rem;
}

.summary-label {
  color: var(--fas-muted);
  font-size: 0.64rem;
}

.summary-value {
  margin-top: 0.45rem;
  color: var(--fas-text);
  font-size: 1.55rem;
  font-weight: 850;
}

.summary-money {
  font-size: 1rem;
}

.summary-priority {
  color: #fbbf24;
}

.summary-helper {
  margin-top: 0.38rem;
  color: #55657d;
  font-size: 0.56rem;
}

.distribution-panel {
  margin-top: 1.2rem;
  overflow: hidden;
  border-radius: 1.15rem;
}

.toolbar {
  display: grid;
  grid-template-columns:
    minmax(280px, 1fr)
    minmax(170px, 0.34fr)
    minmax(190px, 0.38fr);
  gap: 0.7rem;
  padding: 1rem;
  border-bottom: 1px solid var(--fas-border);
}

.search-shell {
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.85rem;
  top: 50%;
  transform: translateY(-50%);
  color: var(--fas-muted);
  font-size: 1rem;
}

.toolbar-input,
.toolbar-select {
  width: 100%;
  min-height: 2.85rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.78rem;
  background: var(--fas-overlay);
  color: var(--fas-text);
  outline: none;
  font-size: 0.68rem;
}

.toolbar-input {
  padding: 0 0.85rem 0 2.55rem;
}

.toolbar-select {
  padding: 0 0.8rem;
  color-scheme: inherit;
}

.toolbar-input:focus,
.toolbar-select:focus {
  border-color: rgba(96, 165, 250, 0.42);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.11);
}

.empty-state {
  padding: 3rem 1rem;
  color: var(--fas-muted);
  text-align: center;
  font-size: 0.72rem;
}

.empty-large {
  padding: 5rem 1.2rem;
}

.empty-large h2 {
  margin-top: 0.8rem;
  color: var(--fas-text-soft);
  font-size: 1rem;
}

.empty-large p {
  max-width: 32rem;
  margin: 0.5rem auto 0;
  line-height: 1.3rem;
}

.empty-icon {
  width: 3rem;
  height: 3rem;
  display: grid;
  place-items: center;
  margin: 0 auto;
  border: 1px solid rgba(96, 165, 250, 0.18);
  border-radius: 999px;
  background: rgba(37, 99, 235, 0.1);
  color: #60a5fa;
  font-size: 1.2rem;
}

.shipment-list {
  display: grid;
}

.shipment-card {
  border-bottom: 1px solid var(--fas-border);
}

.shipment-card:last-child {
  border-bottom: 0;
}

.shipment-main {
  display: grid;
  grid-template-columns:
    minmax(240px, 1.45fr)
    minmax(70px, 0.3fr)
    minmax(130px, 0.55fr)
    minmax(145px, 0.6fr)
    minmax(180px, 0.75fr);
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.1rem;
}

.shipment-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.shipment-title-row h2 {
  color: var(--fas-text);
  font-size: 0.8rem;
  font-weight: 850;
}

.customer-name {
  margin-top: 0.32rem;
  color: var(--fas-text-soft);
  font-size: 0.7rem;
  font-weight: 760;
}

.customer-code {
  margin-top: 0.2rem;
  color: var(--fas-muted);
  font-size: 0.56rem;
}

.status-badge {
  border-radius: 999px;
  padding: 0.25rem 0.48rem;
  font-size: 0.5rem;
  font-weight: 850;
  letter-spacing: 0.05em;
}

.status-packed {
  background: rgba(245, 158, 11, 0.12);
  color: #fbbf24;
}

.status-shipped {
  background: rgba(52, 211, 153, 0.11);
  color: #34d399;
}

.status-neutral {
  background: rgba(148, 163, 184, 0.1);
  color: #94a3b8;
}

.shipment-metric,
.shipment-priority {
  min-width: 0;
}

.shipment-metric span,
.shipment-priority span {
  display: block;
  color: var(--fas-muted);
  font-size: 0.53rem;
  text-transform: uppercase;
  letter-spacing: 0.07em;
}

.shipment-metric strong,
.shipment-priority strong {
  display: block;
  margin-top: 0.28rem;
  color: var(--fas-text);
  font-size: 0.72rem;
  font-weight: 820;
}

.shipment-priority small {
  display: block;
  overflow: hidden;
  margin-top: 0.18rem;
  color: var(--fas-muted);
  font-size: 0.53rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.priority-high-volume {
  color: #fbbf24 !important;
}

.priority-vvip {
  color: #f472b6 !important;
}

.priority-retail {
  color: #60a5fa !important;
}

.priority-regular {
  color: #34d399 !important;
}

.shipment-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
}

.detail-button,
.ship-button {
  min-height: 2.45rem;
  border-radius: 0.7rem;
  padding: 0 0.72rem;
  font-size: 0.6rem;
  font-weight: 820;
  white-space: nowrap;
}

.detail-button {
  border: 1px solid var(--fas-border);
  background: var(--fas-overlay);
  color: var(--fas-text-soft);
}

.detail-button:hover {
  border-color: rgba(96, 165, 250, 0.3);
  color: #dbeafe;
}

.ship-button {
  background: rgba(37, 99, 235, 0.2);
  color: #60a5fa;
}

.ship-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.shipment-detail {
  border-top: 1px solid var(--fas-border);
  background: rgba(6, 12, 24, 0.42);
  padding: 1rem 1.1rem 1.15rem;
}

.detail-loading {
  padding: 1.1rem;
  color: var(--fas-muted);
  text-align: center;
  font-size: 0.64rem;
}

.detail-grid {
  display: grid;
  grid-template-columns:
    1fr
    1fr
    minmax(0, 1.35fr)
    minmax(0, 1.35fr);
  gap: 0.7rem;
}

.detail-box {
  min-width: 0;
  border: 1px solid var(--fas-border);
  border-radius: 0.75rem;
  background: var(--fas-overlay);
  padding: 0.75rem;
}

.detail-box span {
  color: var(--fas-muted);
  font-size: 0.53rem;
  text-transform: uppercase;
  letter-spacing: 0.07em;
}

.detail-box strong {
  display: block;
  margin-top: 0.3rem;
  color: var(--fas-text-soft);
  font-size: 0.66rem;
  line-height: 1.05rem;
}

.detail-box small {
  display: block;
  margin-top: 0.18rem;
  color: var(--fas-muted);
  font-size: 0.54rem;
}

.detail-box-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.detail-edit-button {
  border: 1px solid rgba(96, 165, 250, 0.28);
  border-radius: 0.45rem;
  background: rgba(37, 99, 235, 0.1);
  padding: 0.28rem 0.48rem;
  color: #7cb5ff;
  font-size: 0.54rem;
  font-weight: 800;
  cursor: pointer;
}

.detail-edit-button:hover {
  background: rgba(37, 99, 235, 0.18);
}

.detail-address {
  white-space: pre-line;
  line-height: 0.9rem;
}

.recipient-edit-form {
  display: grid;
  gap: 0.55rem;
  margin-top: 0.6rem;
}

.recipient-edit-form label {
  display: grid;
  gap: 0.3rem;
}

.recipient-edit-form label > span {
  color: var(--fas-muted);
  font-size: 0.51rem;
  font-weight: 750;
  letter-spacing: 0;
  text-transform: none;
}

.recipient-edit-form input,
.recipient-edit-form textarea {
  width: 100%;
  border: 1px solid var(--fas-border);
  border-radius: 0.55rem;
  outline: none;
  background: rgba(7, 14, 26, 0.84);
  padding: 0.58rem 0.65rem;
  color: var(--fas-text);
  font: inherit;
  font-size: 0.62rem;
  box-sizing: border-box;
}

.recipient-edit-form textarea {
  min-height: 4.2rem;
  resize: vertical;
}

.recipient-edit-form input:focus,
.recipient-edit-form textarea:focus {
  border-color: rgba(59, 130, 246, 0.72);
  box-shadow:
    0 0 0 3px
    rgba(37, 99, 235, 0.1);
}

.recipient-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.4rem;
}

.recipient-cancel-button,
.recipient-save-button {
  min-height: 2rem;
  border-radius: 0.5rem;
  padding: 0 0.65rem;
  font-size: 0.56rem;
  font-weight: 800;
  cursor: pointer;
}

.recipient-cancel-button {
  border: 1px solid var(--fas-border);
  background: rgba(15, 23, 42, 0.86);
  color: var(--fas-text-soft);
}

.recipient-save-button {
  border: 1px solid rgba(16, 185, 129, 0.38);
  background: rgba(16, 185, 129, 0.13);
  color: #5ee7b7;
}

.detail-long {
  font-weight: 650 !important;
}

.detail-table-scroll {
  margin-top: 0.8rem;
  overflow-x: auto;
  border: 1px solid var(--fas-border);
  border-radius: 0.75rem;
  scrollbar-width: thin;
  scrollbar-color:
    rgba(71, 85, 105, 0.95)
    rgba(8, 15, 28, 0.96);
}

.detail-table-scroll::-webkit-scrollbar {
  height: 0.48rem;
}

.detail-table-scroll::-webkit-scrollbar-track {
  background: rgba(8, 15, 28, 0.96);
}

.detail-table-scroll::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(71, 85, 105, 0.95);
}

.detail-table {
  width: 100%;
  min-width: 700px;
  border-collapse: collapse;
}

.detail-table th,
.detail-table td {
  border-bottom: 1px solid var(--fas-border);
  padding: 0.72rem 0.8rem;
  text-align: left;
}

.detail-table th {
  color: var(--fas-muted);
  font-size: 0.54rem;
  font-weight: 760;
  text-transform: uppercase;
}

.detail-table td {
  color: var(--fas-text-soft);
  font-size: 0.63rem;
}

.detail-table tbody tr:last-child td {
  border-bottom: 0;
}

.detail-table td strong {
  display: block;
  color: var(--fas-text);
  font-size: 0.64rem;
}

.detail-table td small {
  display: block;
  margin-top: 0.18rem;
  color: var(--fas-muted);
  font-size: 0.52rem;
}

.distribution-note {
  margin-top: 0.8rem;
  border: 1px solid rgba(96, 165, 250, 0.15);
  border-radius: 0.72rem;
  background: rgba(37, 99, 235, 0.07);
  padding: 0.7rem 0.8rem;
  color: #8ea2bd;
  font-size: 0.58rem;
  line-height: 1rem;
}

@container (max-width: 720px) {
  .distribution-archive-info {
    grid-template-columns:
      1fr;
  }

  .distribution-archive-note {
    grid-column:
      auto;
  }
}

@container (max-width: 1080px) {
  .shipment-main {
    grid-template-columns:
      minmax(220px, 1.3fr)
      70px
      minmax(120px, 0.55fr)
      minmax(140px, 0.6fr);
  }

  .shipment-actions {
    grid-column: 1 / -1;
    justify-content: flex-end;
  }

  .detail-grid {
    grid-template-columns:
      repeat(2, minmax(0, 1fr));
  }
}

@container (max-width: 820px) {
  .summary-grid {
    grid-template-columns:
      repeat(2, minmax(0, 1fr));
  }

  .toolbar {
    grid-template-columns: 1fr 1fr;
  }

  .search-shell {
    grid-column: 1 / -1;
  }

  .shipment-main {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(95px, 0.4fr)
      minmax(120px, 0.5fr);
  }

  .shipment-priority {
    grid-column: 1 / 2;
  }

  .shipment-actions {
    grid-column: 2 / -1;
  }
}

@container (max-width: 600px) {
  .distribution-page {
    padding: 1rem;
  }

  .page-header {
    align-items: stretch;
    flex-direction: column;
  }

  .refresh-button {
    width: 100%;
  }

  .summary-grid {
    grid-template-columns:
      repeat(2, minmax(0, 1fr));
  }

  .toolbar {
    grid-template-columns: 1fr;
  }

  .search-shell {
    grid-column: auto;
  }

  .shipment-main {
    grid-template-columns: 1fr;
  }

  .shipment-actions,
  .shipment-priority {
    grid-column: auto;
  }

  .shipment-actions {
    justify-content: stretch;
  }

  .detail-button,
  .ship-button {
    flex: 1;
  }

  .detail-grid {
    grid-template-columns: 1fr;
  }
}

@container (max-width: 420px) {
  .summary-grid {
    grid-template-columns: 1fr;
  }
}

.invoice-button {
  min-height: 2.7rem;
  border: 1px solid rgba(96, 165, 250, 0.28);
  border-radius: 0.72rem;
  background: rgba(37, 99, 235, 0.12);
  padding: 0 0.85rem;
  color: #93c5fd;
  font-size: 0.6rem;
  font-weight: 800;
}

.invoice-button:hover:not(:disabled) {
  background: rgba(37, 99, 235, 0.2);
}

.invoice-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.invoice-print-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  overflow: auto;
  background: #dfe4ea;
  padding: 1rem;
  color: #152033;
}

.invoice-pages-preview {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  padding-bottom: 1.5rem;
}

.invoice-pages-sales .invoice-document {
  width: min(
    100%,
    9.5in
  );
  min-height: 5.5in;
}

.invoice-pages-a4 .invoice-document {
  width: min(
    100%,
    297mm
  );
  min-height: 210mm;
}

.invoice-paper-toolbar-control {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  min-width: 0;
}

.invoice-paper-toolbar-label {
  color: #64748b;
  font-size: 0.62rem;
  font-weight: 750;
  white-space: nowrap;
}

.invoice-paper-segmented {
  display: inline-flex;
  align-items: stretch;
  gap: 0.25rem;
  padding: 0.22rem;
  border: 1px solid #cbd5e1;
  border-radius: 0.7rem;
  background: #eef2f7;
}

.invoice-paper-option {
  min-width: 7.2rem;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 0.08rem;
  border: 1px solid transparent;
  border-radius: 0.52rem;
  background: transparent;
  padding: 0.38rem 0.52rem;
  color: #475569;
  text-align: left;
  cursor: pointer;
}

.invoice-paper-option strong {
  font-size: 0.6rem;
  line-height: 1.05;
}

.invoice-paper-option small {
  color: #7c8ba1;
  font-size: 0.48rem;
  line-height: 1.05;
}

.invoice-paper-option:hover {
  border-color: #bfdbfe;
  background: #eff6ff;
}

.invoice-paper-option-active {
  border-color: #93c5fd;
  background: #dbeafe;
  color: #1d4ed8;
}

.invoice-paper-option-active small {
  color: #3b82f6;
}

.invoice-paper-page-count {
  color: #64748b;
  font-size: 0.56rem;
  font-weight: 700;
  white-space: nowrap;
}

.invoice-continuation-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.4rem;
  border-top: 1px dashed #94a3b8;
  padding: 0.42rem 0.15rem 0;
  color: #64748b;
  font-size: 0.58rem;
}

.invoice-continuation-footer strong {
  color: #334155;
  font-size: 0.56rem;
}

.invoice-print-toolbar {
  width: min(1180px, 100%);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin: 0 auto 0.7rem;
}

.invoice-toolbar-status {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.invoice-toolbar-status strong {
  font-size: 0.78rem;
}

.invoice-toolbar-status span {
  color: #64748b;
  font-size: 0.62rem;
}

.invoice-toolbar-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.45rem;
}

.invoice-toolbar-primary,
.invoice-toolbar-secondary,
.invoice-toolbar-edit,
.invoice-toolbar-save {
  min-height: 2.55rem;
  border-radius: 0.65rem;
  padding: 0 0.9rem;
  font-size: 0.68rem;
  font-weight: 800;
}

.invoice-toolbar-primary {
  border: 1px solid #1d4ed8;
  background: #2563eb;
  color: #fff;
}

.invoice-toolbar-secondary {
  border: 1px solid #cbd5e1;
  background: #fff;
  color: #334155;
}

.invoice-toolbar-edit {
  border: 1px solid #93c5fd;
  background: #eff6ff;
  color: #1d4ed8;
}

.invoice-toolbar-save {
  border: 1px solid #059669;
  background: #059669;
  color: #fff;
}

.invoice-editor-panel {
  width: min(1180px, 100%);
  margin: 0 auto 0.8rem;
  border: 1px solid #cbd5e1;
  border-radius: 0.8rem;
  background: #fff;
  padding: 1rem;
  box-shadow: 0 10px 30px rgba(15, 23, 42, 0.08);
}

.invoice-editor-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid #e2e8f0;
  padding-bottom: 0.7rem;
}

.invoice-editor-heading div {
  display: flex;
  flex-direction: column;
  gap: 0.15rem;
}

.invoice-editor-heading span,
.invoice-editor-heading small {
  color: #64748b;
  font-size: 0.62rem;
}

.invoice-editor-heading strong {
  font-size: 0.82rem;
}

.invoice-editor-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.7rem;
  margin-top: 0.8rem;
}

.invoice-company-address-editor {
  grid-column: 1 / -1;
  display: grid;
  gap: 0.7rem;
  border: 1px solid #d7e0eb;
  border-radius: 0.8rem;
  background: rgba(248, 250, 252, 0.76);
  padding: 0.8rem;
}

.invoice-company-address-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid #e2e8f0;
  padding-bottom: 0.65rem;
}

.invoice-company-address-heading > div {
  display: grid;
  gap: 0.16rem;
}

.invoice-company-address-heading span {
  color: #64748b;
  font-size: 0.55rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-transform: uppercase;
}

.invoice-company-address-heading strong {
  color: #0f172a;
  font-size: 0.72rem;
}

.invoice-company-address-heading small {
  max-width: 19rem;
  color: #64748b;
  font-size: 0.53rem;
  line-height: 0.8rem;
  text-align: right;
}

.invoice-company-fixed-grid {
  display: grid;
  grid-template-columns:
    minmax(0, 1.55fr)
    minmax(0, 0.9fr)
    minmax(0, 1fr);
  gap: 0.55rem;
}

.invoice-company-fixed-item {
  display: grid;
  gap: 0.18rem;
  min-width: 0;
  border: 1px solid #dbe4ef;
  border-radius: 0.65rem;
  background: #f8fafc;
  padding: 0.58rem 0.65rem;
}

.invoice-company-fixed-item span {
  color: #64748b;
  font-size: 0.5rem;
  font-weight: 750;
}

.invoice-company-fixed-item strong {
  overflow: hidden;
  color: #334155;
  font-size: 0.6rem;
  line-height: 0.85rem;
  text-overflow: ellipsis;
}

.invoice-company-address-field {
  margin: 0;
}

.invoice-company-address-field textarea {
  min-height: 5.3rem;
  resize: vertical;
}

.invoice-company-address-field small {
  color: #64748b;
  font-size: 0.53rem;
  line-height: 0.8rem;
}

.invoice-edit-field,
.invoice-editor-item-row label {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.invoice-edit-field > span,
.invoice-editor-item-row label > span {
  color: #64748b;
  font-size: 0.58rem;
  font-weight: 700;
}

.invoice-edit-field input,
.invoice-edit-field select,
.invoice-edit-field textarea,
.invoice-editor-item-row input {
  width: 100%;
  min-height: 2.55rem;
  border: 1px solid #cbd5e1;
  border-radius: 0.55rem;
  background: #f8fafc;
  padding: 0 0.7rem;
  color: #0f172a;
  outline: none;
}

.invoice-edit-field textarea {
  min-height: 4.5rem;
  padding-top: 0.65rem;
  resize: vertical;
}

.invoice-edit-field input:focus,
.invoice-edit-field select:focus,
.invoice-edit-field textarea:focus,
.invoice-editor-item-row input:focus {
  border-color: #60a5fa;
  box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.12);
}

.invoice-edit-number {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 0.45rem;
}

.invoice-edit-number strong {
  font-size: 0.6rem;
}

.invoice-tax-rule-strip {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
  margin-top: 0.75rem;
  border: 1px solid #d7e2ef;
  border-radius: 0.65rem;
  background: #f8fbff;
  padding: 0.7rem 0.8rem;
}

.invoice-tax-rule-strip > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.12rem;
}

.invoice-tax-rule-strip span,
.invoice-tax-rule-strip small {
  color: #64748b;
  font-size: 0.56rem;
}

.invoice-tax-rule-strip strong {
  color: #0f172a;
  font-size: 0.69rem;
}

.invoice-tax-rule-edit {
  flex: 0 0 auto;
  min-height: 2.35rem;
  border: 1px solid #bfdbfe;
  border-radius: 0.55rem;
  background: #eff6ff;
  padding: 0 0.75rem;
  color: #1d4ed8;
  font-size: 0.58rem;
  font-weight: 800;
}

.invoice-tax-rule-editor {
  margin-top: 0.75rem;
  border: 1px solid #bfdbfe;
  border-radius: 0.7rem;
  background: #f8fbff;
  padding: 0.8rem;
}

.invoice-tax-rule-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
  border-bottom: 1px solid #dbeafe;
  padding-bottom: 0.6rem;
}

.invoice-tax-rule-heading > div {
  display: flex;
  flex-direction: column;
  gap: 0.12rem;
}

.invoice-tax-rule-heading span {
  color: #64748b;
  font-size: 0.56rem;
}

.invoice-tax-rule-heading strong {
  color: #0f172a;
  font-size: 0.72rem;
}

.invoice-tax-rule-heading button {
  width: 2rem;
  height: 2rem;
  border: 1px solid #cbd5e1;
  border-radius: 999px;
  background: #fff;
  color: #64748b;
  font-size: 0.85rem;
}

.invoice-tax-rule-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.65rem;
  margin-top: 0.7rem;
}

.invoice-tax-rule-reference {
  grid-column: span 2;
}

.invoice-tax-preview {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 0.12rem;
  min-height: 2.55rem;
  border: 1px solid #cbd5e1;
  border-radius: 0.55rem;
  background: #fff;
  padding: 0.45rem 0.65rem;
}

.invoice-tax-preview span,
.invoice-tax-preview small {
  color: #64748b;
  font-size: 0.52rem;
}

.invoice-tax-preview strong {
  color: #0f172a;
  font-size: 0.76rem;
}

.invoice-tax-rule-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.45rem;
  margin-top: 0.7rem;
}

.invoice-edit-notes {
  margin-top: 0.7rem;
}

.invoice-editor-items {
  margin-top: 0.9rem;
  border-top: 1px solid #e2e8f0;
  padding-top: 0.8rem;
}

.invoice-editor-items-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.6rem;
}

.invoice-editor-items-header strong {
  font-size: 0.7rem;
}

.invoice-editor-items-header span {
  color: #64748b;
  font-size: 0.58rem;
}

.invoice-editor-item-row {
  display: grid;
  grid-template-columns: minmax(14rem, 1.6fr) 1fr 1fr 0.7fr;
  gap: 0.65rem;
  align-items: end;
  border-top: 1px dashed #e2e8f0;
  padding: 0.65rem 0;
}

.invoice-editor-item-name {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  align-self: center;
}

.invoice-editor-item-name strong {
  font-size: 0.68rem;
}

.invoice-editor-item-name span {
  color: #64748b;
  font-size: 0.56rem;
}

.invoice-document {
  width: min(1180px, 100%);
  min-height: 720px;
  margin: 0 auto;
  border: 1px solid #b6c1cd;
  background: #fffef6;
  padding: 1rem 1.05rem;
  box-shadow: 0 18px 60px rgba(15, 23, 42, 0.15);
  color: #1f2937;
  font-family: "Courier New", Courier, monospace;
}

.invoice-original-header {
  display: grid;
  grid-template-columns: 1.15fr 0.9fr 1.15fr;
  gap: 0.55rem;
}

.invoice-company-box,
.invoice-customer-box {
  min-height: 0;
  border: 1px solid #708090;
  padding: 0.38rem 0.52rem;
}

.invoice-company-box,
.invoice-customer-box {
  display: flex;
  flex-direction: column;
  gap: 0.12rem;
}

.invoice-company-box strong,
.invoice-customer-box strong {
  font-size: 0.72rem;
}

.invoice-company-box span,
.invoice-customer-box span {
  font-size: 0.6rem;
  line-height: 1.18;
}

.invoice-customer-address {
  white-space: pre-line;
}

.invoice-middle-meta {
  display: grid;
  grid-template-columns: 1fr 1.25fr;
  align-content: start;
  gap: 0.18rem 0.45rem;
  padding: 0.2rem 0;
}

.invoice-middle-meta > div {
  display: contents;
}

.invoice-middle-meta span {
  font-size: 0.58rem;
}

.invoice-middle-meta strong {
  font-size: 0.61rem;
  font-weight: 700;
}

.invoice-credit-row strong {
  letter-spacing: 0.04em;
}

.invoice-reference-row {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  border: 1px solid #708090;
  border-bottom: 0;
  margin-top: 0.55rem;
}

.invoice-reference-row > div {
  min-height: 2.6rem;
  border-right: 1px solid #708090;
  padding: 0.4rem 0.55rem;
}

.invoice-reference-row > div:last-child {
  border-right: 0;
}

.invoice-reference-row span {
  display: block;
  color: #4b5563;
  font-size: 0.52rem;
}

.invoice-reference-row strong {
  display: block;
  margin-top: 0.15rem;
  font-size: 0.62rem;
}

.invoice-table-scroll {
  overflow-x: auto;
}

.invoice-paper-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.58rem;
}

.invoice-paper-table th,
.invoice-paper-table td {
  border: 1px solid #708090;
  padding: 0.34rem 0.3rem;
  vertical-align: top;
}

.invoice-paper-table th {
  background: rgba(148, 163, 184, 0.08);
  text-align: center;
  white-space: nowrap;
  font-size: 0.53rem;
}

.invoice-paper-table td:not(.invoice-paper-product) {
  text-align: center;
  white-space: nowrap;
}

.invoice-paper-product {
  min-width: 13rem;
}

.invoice-paper-product strong {
  display: block;
  font-size: 0.6rem;
}

.invoice-paper-product small {
  display: block;
  margin-top: 0.1rem;
  color: #64748b;
  font-size: 0.48rem;
}

.invoice-bottom-grid {
  display: grid;
  grid-template-columns: 1fr 22rem;
  margin-top: 0.55rem;
  border: 1px solid #708090;
}

.invoice-signature-box {
  display: grid;
  grid-template-columns: 1fr 1fr;
  min-height: 8.2rem;
  border-right: 1px solid #708090;
}

.invoice-signature-box > div {
  position: relative;
  padding: 0.45rem 0.55rem;
  text-align: center;
}

.invoice-signature-box > div:first-child {
  border-right: 1px solid #708090;
}

.invoice-signature-box span {
  display: block;
  font-size: 0.55rem;
}

.invoice-signature-box strong {
  position: absolute;
  right: 0.45rem;
  bottom: 0.4rem;
  left: 0.45rem;
  border-top: 1px solid #9ca3af;
  padding-top: 0.2rem;
  font-size: 0.52rem;
}

.invoice-tax-box {
  display: flex;
  flex-direction: column;
}

.invoice-tax-box > div {
  display: grid;
  grid-template-columns: 1fr 1fr;
  align-items: center;
  min-height: 1.55rem;
  border-bottom: 1px solid #708090;
  padding: 0.15rem 0.45rem;
  font-size: 0.56rem;
}

.invoice-tax-box > div:last-child {
  border-bottom: 0;
}

.invoice-tax-box strong {
  text-align: right;
}

.invoice-tax-total {
  background: rgba(148, 163, 184, 0.08);
  font-weight: 700;
}

.invoice-paper-footer {
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 0.65rem;
  align-items: center;
  border: 1px solid #708090;
  border-top: 0;
  padding: 0.38rem 0.5rem;
}

.invoice-terbilang {
  min-width: 0;
  font-size: 0.54rem;
  line-height: 1.35;
}

.invoice-terbilang strong {
  margin-right: 0.3rem;
}

.invoice-qr-block {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 10.5rem;
  justify-content: flex-end;
}

.invoice-qr-block img {
  width: 58px;
  height: 58px;
  image-rendering: pixelated;
}

.invoice-qr-block div {
  display: flex;
  flex-direction: column;
  gap: 0.12rem;
  max-width: 8rem;
}

.invoice-qr-block strong {
  font-size: 0.48rem;
  word-break: break-all;
}

.invoice-qr-block span {
  color: #64748b;
  font-size: 0.46rem;
}

.invoice-paper-note {
  margin-top: 0.35rem;
  border: 1px solid #708090;
  padding: 0.35rem 0.45rem;
  font-size: 0.52rem;
}

.invoice-version-note {
  margin-top: 0.25rem;
  text-align: right;
  color: #64748b;
  font-size: 0.46rem;
}

@media (max-width: 860px) {
  .invoice-editor-grid {
    grid-template-columns: 1fr 1fr;
  }

  .invoice-tax-rule-grid {
    grid-template-columns: 1fr 1fr;
  }

  .invoice-editor-item-row {
    grid-template-columns: 1fr 1fr;
  }

  .invoice-original-header {
    grid-template-columns: 1fr;
  }

  .invoice-bottom-grid {
    grid-template-columns: 1fr;
  }

  .invoice-signature-box {
    border-right: 0;
    border-bottom: 1px solid #708090;
  }
}

@media (max-width: 560px) {
  .invoice-print-overlay {
    padding: 0.4rem;
  }

  .invoice-print-toolbar,
  .invoice-editor-heading {
    align-items: stretch;
    flex-direction: column;
  }

  .invoice-toolbar-actions {
    justify-content: stretch;
  }

  .invoice-toolbar-actions button {
    flex: 1 1 auto;
  }

  .invoice-editor-grid,
  .invoice-editor-item-row,
  .invoice-tax-rule-grid {
    grid-template-columns: 1fr;
  }

  .invoice-tax-rule-reference {
    grid-column: span 1;
  }

  .invoice-tax-rule-strip {
    align-items: stretch;
    flex-direction: column;
  }

  .invoice-document {
    padding: 0.55rem;
  }

  .invoice-reference-row {
    grid-template-columns: 1fr;
  }

  .invoice-reference-row > div {
    border-right: 0;
    border-bottom: 1px solid #708090;
  }

  .invoice-reference-row > div:last-child {
    border-bottom: 0;
  }

  .invoice-paper-footer {
    grid-template-columns: 1fr;
  }

  .invoice-qr-block {
    justify-content: flex-start;
  }
}

@media print {
  :global(html),
  :global(body) {
    margin: 0 !important;
    padding: 0 !important;
    overflow: visible !important;
    background: #ffffff !important;
  }

  :global(
    body.fas-invoice-print-mode
    > *:not(.invoice-print-overlay)
  ) {
    display: none !important;
  }

  :global(
    body.fas-invoice-print-mode
    > .invoice-print-overlay
  ) {
    display: block !important;
  }

  .invoice-print-overlay {
    position: static !important;
    inset: auto !important;
    width: auto !important;
    height: auto !important;
    min-width: 0 !important;
    min-height: 0 !important;
    max-width: none !important;
    max-height: none !important;
    margin: 0 !important;
    padding: 0 !important;
    overflow: visible !important;
    background: #ffffff !important;
    color: #111827 !important;
  }

  .invoice-print-toolbar,
  .invoice-editor-panel {
    display: none !important;
  }

  .invoice-pages-preview {
    display: block !important;
    width: auto !important;
    height: auto !important;
    min-height: 0 !important;
    margin: 0 !important;
    padding: 0 !important;
    overflow: visible !important;
  }

  .invoice-document {
    position: relative !important;
    min-height: 0 !important;
    height: auto !important;
    overflow: hidden !important;
    box-sizing: border-box !important;
    background: #fffef6 !important;
    box-shadow: none !important;
    color: #111827 !important;
    font-family:
      "Courier New",
      Courier,
      monospace !important;
    break-inside: avoid-page !important;
    page-break-inside: avoid !important;
    break-after: auto !important;
    page-break-after: auto !important;
  }

  .invoice-document:not(:last-child) {
    break-after: page !important;
    page-break-after: always !important;
  }

  .invoice-document:last-child {
    break-after: auto !important;
    page-break-after: auto !important;
  }

  :global(
    body.fas-invoice-paper-sales
  )
  .invoice-document {
    width: 9.34in !important;
    min-width: 9.34in !important;
    max-width: 9.34in !important;
    max-height: 5.32in !important;
    margin:
      0.035in auto
      0 !important;
    padding: 0.045in 0.06in !important;
    border: 0.009in solid #64748b !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-document {
    width: 288mm !important;
    min-width: 288mm !important;
    max-width: 288mm !important;
    max-height: 201mm !important;
    margin:
      3.5mm auto
      0 !important;
    padding: 3mm 3.5mm !important;
    border: 0.25mm solid #64748b !important;
  }

  .invoice-original-header {
    grid-template-columns:
      1.12fr
      0.98fr
      1.12fr !important;
    gap: 0.035in !important;
    break-inside: avoid !important;
    page-break-inside: avoid !important;
  }

  .invoice-company-box,
  .invoice-customer-box {
    min-height: 0 !important;
    padding: 0.035in 0.045in !important;
    gap: 0.008in !important;
  }

  .invoice-company-box strong,
  .invoice-customer-box strong {
    font-size: 6.15pt !important;
    line-height: 1.02 !important;
  }

  .invoice-company-box span,
  .invoice-customer-box span {
    font-size: 5.15pt !important;
    line-height: 1.03 !important;
  }

  .invoice-middle-meta {
    grid-template-columns:
      0.78fr
      1.22fr !important;
    gap: 0.004in 0.025in !important;
    padding: 0.005in 0 !important;
  }

  .invoice-middle-meta span {
    font-size: 4.95pt !important;
    line-height: 1.01 !important;
  }

  .invoice-middle-meta strong {
    font-size: 5.15pt !important;
    line-height: 1.01 !important;
  }

  .invoice-reference-row {
    margin-top: 0.025in !important;
    break-inside: avoid !important;
    page-break-inside: avoid !important;
  }

  .invoice-reference-row > div {
    min-height: 0 !important;
    padding: 0.018in 0.04in !important;
  }

  .invoice-reference-row span {
    font-size: 4.55pt !important;
    line-height: 1 !important;
  }

  .invoice-reference-row strong {
    margin-top: 0.006in !important;
    font-size: 5.05pt !important;
    line-height: 1 !important;
  }

  .invoice-table-scroll {
    overflow: visible !important;
  }

  .invoice-paper-table {
    width: 100% !important;
    table-layout: fixed !important;
    border-collapse: collapse !important;
    font-size: 4.8pt !important;
  }

  .invoice-paper-table thead {
    display: table-header-group !important;
  }

  .invoice-paper-table tr {
    break-inside: avoid !important;
    page-break-inside: avoid !important;
  }

  .invoice-paper-table th,
  .invoice-paper-table td {
    padding: 0.012in 0.016in !important;
    line-height: 1 !important;
  }

  .invoice-paper-table th {
    font-size: 4.45pt !important;
  }

  .invoice-paper-product {
    min-width: 0 !important;
  }

  .invoice-paper-product strong {
    display: block !important;
    font-size: 4.85pt !important;
    line-height: 1.02 !important;
    overflow-wrap: anywhere !important;
  }

  .invoice-paper-product small {
    display: block !important;
    margin-top: 0.003in !important;
    font-size: 3.75pt !important;
    line-height: 1 !important;
    overflow-wrap: anywhere !important;
  }

  .invoice-continuation-footer {
    margin-top: 0.015in !important;
    border-top: 0.008in dashed #94a3b8 !important;
    padding: 0.012in 0.015in 0 !important;
    font-size: 4.1pt !important;
  }

  .invoice-continuation-footer strong {
    font-size: 4.05pt !important;
  }

  .invoice-bottom-grid {
    grid-template-columns:
      minmax(0, 1fr)
      2.24in !important;
    margin-top: 0.02in !important;
    break-inside: avoid !important;
    page-break-inside: avoid !important;
  }

  .invoice-signature-box {
    min-height: 0.56in !important;
  }

  .invoice-signature-box > div {
    padding: 0.025in 0.04in !important;
  }

  .invoice-signature-box span {
    font-size: 4.55pt !important;
  }

  .invoice-signature-box strong {
    right: 0.04in !important;
    bottom: 0.02in !important;
    left: 0.04in !important;
    padding-top: 0.01in !important;
    font-size: 4.35pt !important;
  }

  .invoice-tax-box > div {
    min-height: 0.11in !important;
    padding: 0.006in 0.03in !important;
    font-size: 4.45pt !important;
    line-height: 1 !important;
  }

  .invoice-paper-footer {
    grid-template-columns:
      minmax(0, 1fr)
      auto !important;
    gap: 0.03in !important;
    padding: 0.018in 0.035in !important;
    break-inside: avoid !important;
    page-break-inside: avoid !important;
  }

  .invoice-terbilang {
    font-size: 4.35pt !important;
    line-height: 1.04 !important;
  }

  .invoice-qr-block {
    min-width: 0.86in !important;
    gap: 0.02in !important;
  }

  .invoice-qr-block img {
    width: 0.4in !important;
    height: 0.4in !important;
  }

  .invoice-qr-block div {
    max-width: 0.72in !important;
    gap: 0.003in !important;
  }

  .invoice-qr-block strong {
    font-size: 3.55pt !important;
    line-height: 1 !important;
  }

  .invoice-qr-block span {
    font-size: 3.4pt !important;
    line-height: 1 !important;
  }

  .invoice-paper-note {
    margin-top: 0.012in !important;
    padding: 0.014in 0.03in !important;
    font-size: 4.15pt !important;
    line-height: 1.02 !important;
    break-inside: avoid !important;
    page-break-inside: avoid !important;
  }

  .invoice-version-note {
    display: block !important;
    margin-top: 0.004in !important;
    font-size: 3.55pt !important;
    line-height: 1 !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-company-box strong,
  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-customer-box strong {
    font-size: 7.7pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-company-box span,
  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-customer-box span {
    font-size: 6.5pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-middle-meta span {
    font-size: 6.1pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-middle-meta strong {
    font-size: 6.45pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-reference-row span {
    font-size: 5.8pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-reference-row strong {
    font-size: 6.3pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-table {
    font-size: 6.2pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-table th,
  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-table td {
    padding: 1mm 1.05mm !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-table th {
    font-size: 5.7pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-product strong {
    font-size: 6.25pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-product small {
    font-size: 4.9pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-bottom-grid {
    grid-template-columns:
      minmax(0, 1fr)
      68mm !important;
    margin-top: 1.4mm !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-signature-box {
    min-height: 25mm !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-tax-box > div {
    min-height: 4.4mm !important;
    padding: 0.4mm 1.1mm !important;
    font-size: 5.7pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-terbilang {
    font-size: 5.7pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-qr-block img {
    width: 15mm !important;
    height: 15mm !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-paper-note {
    font-size: 5.4pt !important;
  }

  :global(
    body.fas-invoice-paper-a4
  )
  .invoice-version-note {
    font-size: 4.5pt !important;
  }
}


.invoice-warehouse-class {
  color: #64748b;
  font-size: 0.48rem !important;
  letter-spacing: 0.03em;
}



  :global(
    body.fas-invoice-paper-sales
  )
  .invoice-bottom-grid {
    margin-top: 0.03in !important;
  }

  :global(
    body.fas-invoice-paper-sales
  )
  .invoice-signature-box {
    min-height: 0.68in !important;
  }

  :global(
    body.fas-invoice-paper-sales
  )
  .invoice-paper-footer {
    padding:
      0.025in
      0.04in !important;
  }

  :global(
    body.fas-invoice-paper-sales
  )
  .invoice-version-note {
    margin-top: 0.008in !important;
  }

@media (max-width: 980px) {
  .invoice-paper-toolbar-control {
    width: 100%;
    flex-wrap: wrap;
  }

  .invoice-paper-segmented {
    flex: 1;
  }

  .invoice-paper-option {
    min-width: 0;
    flex: 1;
  }
}


@media (max-width: 860px) {
  .invoice-company-address-heading {
    flex-direction: column;
  }

  .invoice-company-address-heading small {
    max-width: none;
    text-align: left;
  }

  .invoice-company-fixed-grid {
    grid-template-columns: 1fr;
  }
}


.invoice-editor-panel {
  border-color: var(--fas-border-strong);
  background: var(--fas-card);
  color: var(--fas-text);
}

.invoice-editor-heading,
.invoice-editor-item-name strong {
  color: var(--fas-text);
}

.invoice-edit-field > span,
.invoice-editor-item-row label > span {
  color: var(--fas-text-soft);
  font-size: 0.75rem;
}

.invoice-edit-field input,
.invoice-edit-field textarea,
.invoice-editor-item-row input {
  min-height: 2.85rem;
  border-color: var(--fas-border-strong);
  border-radius: 0.85rem;
  background: var(--fas-input);
  color: var(--fas-text);
  font-size: 0.85rem;
}

.invoice-edit-field input:focus,
.invoice-edit-field textarea:focus,
.invoice-editor-item-row input:focus {
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.refresh-icon,
.search-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.refresh-button:hover:not(:disabled) {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.toolbar-input {
  font-size: 0.85rem;
}

</style>
