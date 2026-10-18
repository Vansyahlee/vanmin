<script setup>
import FasIcon from '@/components/ui/FasIcon.vue'
import { useModalDialog } from '@/composables/useModalDialog'

import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
} from 'vue'

import { useRouter } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { getProducts } from '@/services/product'
import { getCustomers, createCustomer } from '@/services/customer'
import {
  getSalesOrders,
  getSalesOrderByID,
  createSalesOrder,
  confirmSalesOrder,
  startSalesOrderPicking,
  finishSalesOrderPacking,
} from '@/services/salesOrder'

const authStore = useAuthStore()
const router = useRouter()

const loading = ref(true)
const customerLoading = ref(true)
const customerCreating = ref(false)
const saving = ref(false)
const transitioning = ref(false)

const operationsRoot = ref(null)
const compactWorkspace = ref(false)
let workspaceObserver = null

const showSidePanel = ref(true)
const showDSSPanel = ref(true)
const showWorkflowPanel = ref(true)

const products = ref([])
const customers = ref([])

const productSearch = ref('')
const customerSearch = ref('')

const selectedProductID = ref('')
const selectedQuantity = ref(1)

const productDropdownOpen = ref(false)
const customerDropdownOpen = ref(false)

const successMessage = ref('')
const errorMessage = ref('')

const savedOrder = ref(null)
const savedSignature = ref('')

const order = reactive({
  customer_id: '',
  store_name: '',
  store_code: '',
  address: '',
  phone: '',
  order_date: new Date().toISOString().slice(0, 10),
  notes: '',
  items: [],
})

const currentRole = computed(() => {
  return String(
    authStore.user?.role || '',
  ).toLowerCase()
})

const canManageOrder = computed(() => {
  return [
    'admin',
    'owner',
    'administrator',
    'pegawai',
    'warehouse',
    'gudang',
  ].includes(currentRole.value)
})

const sidePanelsVisible = computed(() => {
  return (
    showSidePanel.value &&
    (
      showDSSPanel.value ||
      showWorkflowPanel.value
    )
  )
})

const toggleSidePanel = () => {
  if (sidePanelsVisible.value) {
    showSidePanel.value = false
  } else {
    showSidePanel.value = true
    showDSSPanel.value = true
    showWorkflowPanel.value = true
  }
}

const closeSidePanel = () => {
  showSidePanel.value = false
}

const activeCustomers = computed(() => {
  return customers.value.filter(
    (customer) => customer.is_active,
  )
})

const filteredCustomers = computed(() => {
  const keyword =
    customerSearch.value
      .trim()
      .toLowerCase()

  if (!keyword) {
    return activeCustomers.value
  }

  return activeCustomers.value.filter(
    (customer) => {
      const source = [
        customer.customer_code,
        customer.name,
        customer.phone,
        customer.email,
        customer.shipping_address,
        customer.tax_id_number,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()

      return source.includes(keyword)
    },
  )
})

const exactCustomerMatch = computed(() => {
  const keyword =
    customerSearch.value
      .trim()
      .toLowerCase()

  if (!keyword) {
    return null
  }

  return activeCustomers.value.find(
    (customer) => {
      const name =
        String(
          customer.name || '',
        )
          .trim()
          .toLowerCase()

      const code =
        String(
          customer.customer_code || '',
        )
          .trim()
          .toLowerCase()

      return (
        name === keyword ||
        code === keyword
      )
    },
  ) || null
})

const canCreateCustomerFromSearch = computed(() => {
  return (
    canManageOrder.value &&
    customerSearch.value
      .trim()
      .length >= 3 &&
    !exactCustomerMatch.value
  )
})

const selectedCustomer = computed(() => {
  return activeCustomers.value.find(
    (customer) =>
      customer.id === order.customer_id,
  ) || null
})

const paymentTermsOptions = [
  {
    value: 14,
    label: '14 HARI',
  },
  {
    value: 21,
    label: '21 HARI',
  },
  {
    value: 30,
    label: '30 HARI',
  },
]

const paymentTermsMode = ref('')
const manualPaymentTermsOpen = ref(false)
const manualPaymentTermsDays = ref('')
const paymentTermsDropdownOpen = ref(false)

const paymentTermsDays = computed(() => {
  if (paymentTermsMode.value === 'MANUAL') {
    const days =
      Number(
        manualPaymentTermsDays.value,
      )

    if (
      !Number.isFinite(days) ||
      days <= 0
    ) {
      return null
    }

    return Math.trunc(days)
  }

  if (paymentTermsMode.value === '') {
    return null
  }

  return Number(
    paymentTermsMode.value,
  )
})

const paymentTermsLabel = computed(() => {
  if (paymentTermsDays.value === null) {
    return 'Pilih Tempo'
  }

  return `${paymentTermsDays.value} HARI`
})

const activeProducts = computed(() => {
  return products.value.filter(
    (product) => product.is_active,
  )
})

const filteredProducts = computed(() => {
  const keyword =
    productSearch.value
      .trim()
      .toLowerCase()

  if (!keyword) {
    return activeProducts.value
  }

  return activeProducts.value.filter(
    (product) => {
      const source = [
        product.name,
        product.product_code,
        product.sku,
        product.barcode,
        product.unit,
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase()

      return source.includes(keyword)
    },
  )
})

const selectedProduct = computed(() => {
  return activeProducts.value.find(
    (product) =>
      product.id ===
      selectedProductID.value,
  ) || null
})

const totalQty = computed(() => {
  return order.items.reduce(
    (total, item) =>
      total + Number(item.qty || 0),
    0,
  )
})

const subtotal = computed(() => {
  return order.items.reduce(
    (total, item) =>
      total +
      Number(item.price || 0) *
        Number(item.qty || 0),
    0,
  )
})

const buildDraftPayload = () => {
  return {
    customer_id:
      order.customer_id,
    order_date:
      order.order_date,
    notes:
      order.notes.trim(),
    payment_terms_days:
      Number(
        paymentTermsDays.value ||
        0,
      ),
    discount_amount: 0,
    tax_scheme: 'NON_PPN',
    items:
      order.items.map(
        (item) => ({
          product_id:
            item.product_id,
          quantity:
            Number(item.qty),
        }),
      ),
  }
}

const currentDraftSignature = computed(() => {
  return JSON.stringify(
    buildDraftPayload(),
  )
})

const backendResultCurrent = computed(() => {
  return Boolean(
    savedOrder.value &&
    savedSignature.value &&
    savedSignature.value ===
      currentDraftSignature.value,
  )
})

const currentOrderStatus = computed(() => {
  return String(
    savedOrder.value?.status ||
    'DRAFT',
  ).toUpperCase()
})

const workflowStatuses = [
  'DRAFT',
  'CONFIRMED',
  'PICKING',
  'PACKED',
  'SHIPPED',
]

const getOrderStatusLabel = (status) => {
  switch (
    String(
      status || '',
    ).toUpperCase()
  ) {
  case 'CONFIRMED':
    return 'DIKONFIRMASI'
  case 'PICKING':
    return 'PENGAMBILAN'
  case 'PACKED':
    return 'DIKEMAS'
  case 'SHIPPED':
    return 'DIKIRIM'
  case 'COMPLETED':
    return 'SELESAI'
  case 'CANCELLED':
    return 'DIBATALKAN'
  default:
    return 'DRAFT'
  }
}

const workflowCurrentIndex = computed(() => {
  const index =
    workflowStatuses.indexOf(
      currentOrderStatus.value,
    )

  return index < 0
    ? 0
    : index
})

const workflowAction = computed(() => {
  if (
    !backendResultCurrent.value ||
    !savedOrder.value?.id
  ) {
    return null
  }

  switch (currentOrderStatus.value) {
  case 'DRAFT':
    return {
      label: 'Konfirmasi Pesanan',
      next: 'CONFIRMED',
    }

  case 'CONFIRMED':
    return {
      label: 'Mulai Pengambilan',
      next: 'PICKING',
    }

  case 'PICKING':
    return {
      label: 'Selesai Packing',
      next: 'PACKED',
    }

  default:
    return null
  }
})

const workflowItemClass = (status) => {
  const index =
    workflowStatuses.indexOf(
      status,
    )

  return {
    active:
      index ===
      workflowCurrentIndex.value,
    completed:
      index <
      workflowCurrentIndex.value,
  }
}

const previewDecisionTree = (
  jumlah,
  clQty,
) => {
  if (clQty <= 18.5) {
    if (jumlah <= 605117) {
      return 'Klaster 1 (Retail)'
    }

    if (clQty <= 12.5) {
      if (jumlah <= 1177470) {
        return 'Klaster 1 (Retail)'
      }

      return 'Klaster 0 (Reguler)'
    }

    return 'Klaster 0 (Reguler)'
  }

  if (clQty <= 75.5) {
    return 'Klaster 0 (Reguler)'
  }

  if (jumlah <= 1723267) {
    if (jumlah <= 1377936) {
      return 'Klaster 2 (VVIP)'
    }

    return 'Klaster 0 (Reguler)'
  }

  return 'Klaster 2 (VVIP)'
}

const mapDSSResult = (
  cluster,
  recommendation = '',
  priority = '',
  clQty = 0,
) => {
  if (
    cluster.includes('VVIP') ||
    priority === 'VVIP'
  ) {
    return {
      cluster,
      shortLabel: 'VVIP',
      priority: 'VVIP',
      recommendation:
        recommendation ||
        'PRIORITAS EKSPRES! Gunakan jalur utama & pastikan Safety Stock 100%.',
      className: 'dss-vvip',
    }
  }

  if (
    priority === 'HIGH_VOLUME' ||
    (
      Number(clQty) >= 60 &&
      !cluster.includes('VVIP')
    )
  ) {
    return {
      cluster,
      shortLabel: 'HIGH VOLUME',
      priority: 'HIGH_VOLUME',
      recommendation:
        recommendation ||
        'Pesanan volume besar. Prioritaskan kapasitas picking, packing, dan ruang kendaraan distribusi.',
      className: 'dss-high-volume',
    }
  }

  if (
    cluster.includes('Retail') ||
    priority === 'RETAIL'
  ) {
    return {
      cluster,
      shortLabel: 'RETAIL',
      priority: 'RETAIL',
      recommendation:
        recommendation ||
        'Gabungkan rute pengiriman (Milk Run) untuk efisiensi BBM.',
      className: 'dss-retail',
    }
  }

  return {
    cluster,
    shortLabel: 'REGULER',
    priority: 'REGULAR',
    recommendation:
      recommendation ||
      'Jalur distribusi reguler standar.',
    className: 'dss-regular',
  }
}

const dssResult = computed(() => {
  if (backendResultCurrent.value) {
    return mapDSSResult(
      savedOrder.value.cluster ||
        'Klaster 0 (Reguler)',
      savedOrder.value.recommendation ||
        '',
      savedOrder.value.priority ||
        '',
      savedOrder.value.cl_qty ||
        totalQty.value,
    )
  }

  if (
    subtotal.value > 0 &&
    totalQty.value > 0
  ) {
    return mapDSSResult(
      previewDecisionTree(
        subtotal.value,
        totalQty.value,
      ),
      '',
      '',
      totalQty.value,
    )
  }

  return {
    cluster: 'BELUM DIANALISIS',
    shortLabel: '—',
    priority: '—',
    recommendation:
      'Pilih customer dan tambahkan produk ke pesanan.',
    className: 'dss-neutral',
  }
})

const hasInsufficientStock = computed(() => {
  return order.items.some(
    (item) =>
      Number(item.qty || 0) >
      Number(item.stock || 0),
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

const clearMessage = () => {
  successMessage.value = ''
  errorMessage.value = ''
}

const loadProducts = async () => {
  loading.value = true

  try {
    products.value =
      await getProducts()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Data produk tidak dapat dimuat. Silakan coba kembali.'
  } finally {
    loading.value = false
  }
}

const loadCustomers = async () => {
  customerLoading.value = true

  try {
    customers.value =
      await getCustomers()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Data customer tidak dapat dimuat. Silakan coba kembali.'
  } finally {
    customerLoading.value = false
  }
}

const closeDropdowns = () => {
  productDropdownOpen.value = false
  customerDropdownOpen.value = false
  paymentTermsDropdownOpen.value = false
}

const toggleCustomerDropdown = () => {
  customerDropdownOpen.value =
    !customerDropdownOpen.value

  productDropdownOpen.value = false
}

const toggleProductDropdown = () => {
  productDropdownOpen.value =
    !productDropdownOpen.value

  customerDropdownOpen.value = false
}

const createCustomerFromSearch = async () => {
  clearMessage()

  const name =
    customerSearch.value.trim()

  if (!name) {
    return
  }

  if (exactCustomerMatch.value) {
    selectCustomer(
      exactCustomerMatch.value,
    )
    return
  }

  if (!canManageOrder.value) {
    errorMessage.value =
      'Anda tidak memiliki izin untuk menambahkan customer baru.'
    return
  }

  if (name.length < 3) {
    errorMessage.value =
      'Nama customer minimal 3 karakter.'
    return
  }

  customerCreating.value = true

  try {
    const created =
      await createCustomer({
        name,
        phone: '',
        email: '',
        shipping_address: '',
        tax_id_type: '',
        tax_id_number: '',
        tax_name: '',
        tax_address: '',
        is_pkp: false,
        tku_id: '',
        payment_terms_days: 14,
      })

    customers.value = [
      created,
      ...customers.value.filter(
        (customer) =>
          customer.id !==
          created.id,
      ),
    ]

    selectCustomer(created)

    successMessage.value =
      `${created.customer_code} · ${created.name} berhasil didaftarkan. Lengkapi informasi pesanan untuk melanjutkan.`
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Customer baru tidak dapat didaftarkan. Silakan coba kembali.'
  } finally {
    customerCreating.value = false
  }
}

const togglePaymentTermsDropdown = () => {
  paymentTermsDropdownOpen.value =
    !paymentTermsDropdownOpen.value

  customerDropdownOpen.value = false
  productDropdownOpen.value = false

  if (!paymentTermsDropdownOpen.value) {
    manualPaymentTermsOpen.value = false
  }
}

const selectPaymentTerms = (value) => {
  paymentTermsMode.value =
    String(value)

  manualPaymentTermsOpen.value = false
  manualPaymentTermsDays.value = ''
  paymentTermsDropdownOpen.value = false

  clearMessage()
}

const openManualPaymentTerms = () => {
  paymentTermsMode.value = 'MANUAL'
  manualPaymentTermsOpen.value = true
  manualPaymentTermsDays.value = ''
  clearMessage()
}

const commitManualPaymentTerms = () => {
  const days =
    Number.parseInt(
      String(
        manualPaymentTermsDays.value ||
        '',
      ),
      10,
    )

  if (
    !Number.isFinite(days) ||
    days <= 0
  ) {
    errorMessage.value =
      'Tempo manual harus lebih dari 0 hari.'
    return
  }

  manualPaymentTermsDays.value =
    String(
      Math.min(days, 365),
    )

  paymentTermsMode.value = 'MANUAL'
  manualPaymentTermsOpen.value = false
  paymentTermsDropdownOpen.value = false

  clearMessage()
}

const selectCustomer = (customer) => {
  order.customer_id =
    customer.id
  order.store_name =
    customer.name || ''
  order.store_code =
    customer.customer_code || ''
  order.address =
    customer.shipping_address || ''
  order.phone =
    customer.phone || ''

  customerSearch.value = ''
  customerDropdownOpen.value = false

  clearMessage()
}

const selectProduct = (product) => {
  selectedProductID.value =
    product.id

  productDropdownOpen.value = false
  clearMessage()
}

const addProduct = () => {
  clearMessage()

  if (!selectedProduct.value) {
    errorMessage.value =
      'Pilih produk terlebih dahulu.'
    return
  }

  const qty =
    Number(selectedQuantity.value)

  if (
    !Number.isInteger(qty) ||
    qty <= 0
  ) {
    errorMessage.value =
      'Jumlah pesanan harus lebih dari 0.'
    return
  }

  const product =
    selectedProduct.value

  const existing =
    order.items.find(
      (item) =>
        item.product_id ===
        product.id,
    )

  if (existing) {
    existing.qty += qty
  } else {
    order.items.push({
      product_id: product.id,
      product_code:
        product.product_code,
      sku: product.sku,
      name: product.name,
      unit: product.unit,
      price: Number(
        product.price || 0,
      ),
      stock: Number(
        product.stock || 0,
      ),
      qty,
    })
  }

  selectedProductID.value = ''
  selectedQuantity.value = 1
  productSearch.value = ''
}

const removeItem = (productID) => {
  order.items =
    order.items.filter(
      (item) =>
        item.product_id !==
        productID,
    )

  clearMessage()
}

const normalizeQty = (value) => {
  const qty =
    Number.parseInt(
      String(value || 1),
      10,
    )

  if (
    !Number.isFinite(qty) ||
    qty < 1
  ) {
    return 1
  }

  return qty
}

const decreaseSelectedQuantity = () => {
  selectedQuantity.value =
    Math.max(
      1,
      normalizeQty(
        selectedQuantity.value,
      ) - 1,
    )

  clearMessage()
}

const increaseSelectedQuantity = () => {
  selectedQuantity.value =
    normalizeQty(
      selectedQuantity.value,
    ) + 1

  clearMessage()
}

const decreaseItemQty = (item) => {
  item.qty =
    Math.max(
      1,
      normalizeQty(item.qty) - 1,
    )

  clearMessage()
}

const increaseItemQty = (item) => {
  item.qty =
    normalizeQty(item.qty) + 1

  clearMessage()
}

const updateQty = (item) => {
  item.qty =
    normalizeQty(item.qty)

  clearMessage()
}

const resetOrder = () => {
  clearMessage()

  order.customer_id = ''
  order.store_name = ''
  order.store_code = ''
  order.address = ''
  order.phone = ''
  order.order_date =
    new Date()
      .toISOString()
      .slice(0, 10)
  order.notes = ''
  order.items = []

  selectedProductID.value = ''
  selectedQuantity.value = 1
  productSearch.value = ''
  customerSearch.value = ''

  paymentTermsMode.value = ''
  manualPaymentTermsOpen.value = false
  manualPaymentTermsDays.value = ''
  paymentTermsDropdownOpen.value = false

  savedOrder.value = null
  savedSignature.value = ''

  localStorage.removeItem(
    'fas-operation-order-draft',
  )
}

const validateDraft = () => {
  if (!order.customer_id) {
    return 'Pilih customer dari Customer Master.'
  }

  if (!selectedCustomer.value) {
    return 'Customer tidak valid atau sedang nonaktif.'
  }

  if (!order.order_date) {
    return 'Tanggal pesanan wajib diisi.'
  }

  if (paymentTermsDays.value === null) {
    return 'Pilih tempo pembayaran.'
  }

  if (order.items.length === 0) {
    return 'Tambahkan minimal satu produk.'
  }

  for (const item of order.items) {
    const qty = Number(item.qty)

    if (
      !Number.isInteger(qty) ||
      qty <= 0
    ) {
      return `Qty ${item.name} tidak valid.`
    }
  }

  return null
}

const saveDraft = async () => {
  clearMessage()

  const validationError =
    validateDraft()

  if (validationError) {
    errorMessage.value =
      validationError
    return
  }

  if (
    backendResultCurrent.value
  ) {
    successMessage.value =
      `${savedOrder.value.order_number} sudah tersimpan sebagai draft.`
    return
  }

  saving.value = true

  try {
    const payload =
      buildDraftPayload()

    const result =
      await createSalesOrder(
        payload,
      )

    savedOrder.value = result
    savedSignature.value =
      JSON.stringify(payload)

    localStorage.removeItem(
      'fas-operation-order-draft',
    )

    const shortageMessage =
      result.has_stock_shortage
        ? ' Ada item yang melebihi stok dan ditandai untuk tindak lanjut.'
        : ''

    successMessage.value =
      `${result.order_number} berhasil disimpan. Klasifikasi: ${result.cluster}.${shortageMessage}`
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Pesanan tidak dapat disimpan. Silakan coba kembali.'
  } finally {
    saving.value = false
  }
}


const restoreLatestOpenOrder = async () => {
  try {
    const allOrders =
      await getSalesOrders()

    const openOrder =
      allOrders.find(
        (item) =>
          [
            'DRAFT',
            'CONFIRMED',
            'PICKING',
          ].includes(
            String(
              item.status || '',
            ).toUpperCase(),
          ),
      )

    if (!openOrder?.id) {
      return
    }

    const detail =
      await getSalesOrderByID(
        openOrder.id,
      )

    if (!detail) {
      return
    }

    order.customer_id =
      detail.customer_id || ''
    order.store_name =
      detail.customer_name || ''
    order.store_code =
      detail.customer_code || ''
    order.order_date =
      String(
        detail.order_date || '',
      ).slice(0, 10)
    order.notes =
      detail.notes || ''

    const customer =
      customers.value.find(
        (item) =>
          item.id ===
          detail.customer_id,
      )

    order.phone =
      customer?.phone || ''
    order.address =
      customer?.shipping_address || ''

    const terms =
      Number(
        detail.payment_terms_days ||
        0,
      )

    if (
      [14, 21, 30].includes(
        terms,
      )
    ) {
      paymentTermsMode.value =
        String(terms)
      manualPaymentTermsDays.value = ''
    } else if (terms > 0) {
      paymentTermsMode.value =
        'MANUAL'
      manualPaymentTermsDays.value =
        String(terms)
    } else {
      paymentTermsMode.value = ''
      manualPaymentTermsDays.value = ''
    }

    order.items =
      (detail.items || []).map(
        (item) => {
          const product =
            products.value.find(
              (productItem) =>
                productItem.id ===
                item.product_id,
            )

          return {
            product_id:
              item.product_id,
            product_code:
              item.product_code,
            sku:
              item.sku,
            barcode:
              item.barcode,
            name:
              item.product_name,
            unit:
              item.unit,
            price:
              Number(
                item.unit_price ||
                0,
              ),
            stock:
              Number(
                product?.stock ||
                0,
              ),
            qty:
              Number(
                item.quantity ||
                0,
              ),
          }
        },
      )

    savedOrder.value = detail
    savedSignature.value =
      JSON.stringify(
        buildDraftPayload(),
      )

    successMessage.value =
      `${detail.order_number} siap dilanjutkan. Status: ${getOrderStatusLabel(detail.status)}.`
  } catch (error) {
    console.error(
      'Pesanan aktif tidak dapat dimuat:',
      error,
    )
  }
}

const advanceWorkflow = async () => {
  clearMessage()

  if (
    !savedOrder.value?.id ||
    !workflowAction.value
  ) {
    return
  }

  transitioning.value = true

  try {
    let result = null

    switch (
      currentOrderStatus.value
    ) {
    case 'DRAFT':
      result =
        await confirmSalesOrder(
          savedOrder.value.id,
        )
      break

    case 'CONFIRMED':
      result =
        await startSalesOrderPicking(
          savedOrder.value.id,
        )
      break

    case 'PICKING':
      result =
        await finishSalesOrderPacking(
          savedOrder.value.id,
        )
      break

    default:
      return
    }

    savedOrder.value = result

    if (
      result.status ===
      'PACKED'
    ) {
      successMessage.value =
        `${result.order_number} selesai dikemas dan siap diproses untuk pengiriman.`

      await router.push(
        '/distribution',
      )
    } else {
      successMessage.value =
        `${result.order_number} berhasil diperbarui ke status ${getOrderStatusLabel(result.status)}.`
    }
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Status pesanan tidak dapat diperbarui. Silakan coba kembali.'
  } finally {
    transitioning.value = false
  }
}

useModalDialog(
  () => compactWorkspace.value && sidePanelsVisible.value ? '.side-panel-drawer' : null,
  closeSidePanel,
)

onMounted(async () => {
  workspaceObserver = new ResizeObserver(entries => {
    const compact = entries[0].contentRect.width <= 680

    if (compact && !compactWorkspace.value) {
      showSidePanel.value = false
    }

    compactWorkspace.value = compact
  })

  if (operationsRoot.value) {
    workspaceObserver.observe(operationsRoot.value)
  }

  document.addEventListener(
    'click',
    closeDropdowns,
  )

  await Promise.all([
    loadProducts(),
    loadCustomers(),
  ])

  await restoreLatestOpenOrder()
})

onBeforeUnmount(() => {
  workspaceObserver?.disconnect()
  document.removeEventListener(
    'click',
    closeDropdowns,
  )
})
</script>

<template>
  <div ref="operationsRoot" class="operations-page" @keydown.esc="closeDropdowns">
    <header class="page-header">
      <div>
        <p class="eyebrow">
          Operations Management
        </p>

        <h1 class="page-title">
          Pesanan Toko
        </h1>

        <p class="page-description">
          Input pesanan customer, pilih produk, hitung nilai transaksi,
          dan tentukan prioritas distribusi.
        </p>
      </div>

      <div class="header-actions">
        <div class="view-panel-actions">
          <button
            type="button"
            :class="[
              'panel-toggle-button',
              sidePanelsVisible
                ? 'panel-toggle-active'
                : '',
            ]"
            :aria-expanded="sidePanelsVisible"
            aria-controls="operations-side-panel"
            @click="toggleSidePanel"
          >
            <span class="side-panel-icon" aria-hidden="true">
              <FasIcon name="panel-right" />
            </span>

            {{
              sidePanelsVisible
                ? 'Sembunyikan Panel'
                : 'Tampilkan Panel'
            }}
          </button>
        </div>

        <button
          type="button"
          class="secondary-button"
          @click="resetOrder"
        >
          Reset
        </button>

        <button
          type="button"
          class="primary-button"
          :disabled="
            !canManageOrder ||
            saving ||
            backendResultCurrent
          "
          @click="saveDraft"
        >
          {{
            saving
              ? 'Menyimpan...'
              : backendResultCurrent
                ? 'Draft Tersimpan'
                : 'Simpan Draft'
          }}
        </button>

        <button
          v-if="workflowAction"
          type="button"
          class="workflow-action-button"
          :disabled="
            !canManageOrder ||
            transitioning
          "
          @click="advanceWorkflow"
        >
          {{
            transitioning
              ? 'Memproses...'
              : workflowAction.label
          }}
        </button>

        <button
          v-else-if="
            backendResultCurrent &&
            currentOrderStatus ===
            'PACKED'
          "
          type="button"
          class="workflow-ready-button"
          disabled
        >
          Siap Dikirim
        </button>
      </div>
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
          Total Item
        </p>

        <p class="summary-value">
          {{ order.items.length }}
        </p>

        <p class="summary-helper">
          Jenis produk dalam pesanan
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Total Qty
        </p>

        <p class="summary-value">
          {{ formatNumber(totalQty) }}
        </p>

        <p class="summary-helper">
          Total kuantitas pesanan
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Nilai Pesanan
        </p>

        <p class="summary-value summary-money">
          {{ formatCurrency(subtotal) }}
        </p>

        <p class="summary-helper">
          Nilai transaksi
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Prioritas
        </p>

        <p
          :class="[
            'summary-value',
            'summary-priority',
            dssResult.className,
          ]"
        >
          {{ dssResult.shortLabel }}
        </p>

        <p class="summary-helper">
          Prioritas distribusi
        </p>
      </article>
    </section>

    <div
      :class="[
        'workspace-grid',
        !sidePanelsVisible
          ? 'workspace-full'
          : '',
      ]"
    >
      <main class="main-column">
        <section class="panel">
          <div class="panel-header">
            <div>
              <p class="panel-kicker">
                Customer
              </p>

              <h2 class="panel-title">
                Informasi Pesanan
              </h2>
            </div>

            <span
              :class="[
                'draft-badge',
                currentOrderStatus !==
                'DRAFT'
                  ? 'draft-badge-progress'
                  : '',
              ]"
            >
              {{
                getOrderStatusLabel(
                  currentOrderStatus,
                )
              }}
            </span>
          </div>

          <div class="form-grid">
            <div
              class="field span-two"
              @click.stop
            >
              <label class="field-label">
                Customer Terdaftar
              </label>

              <div class="fas-select-shell">
                <button
                  type="button"
                  class="fas-select-trigger"
                  :disabled="customerLoading"
                  @click="toggleCustomerDropdown"
                >
                  <span>
                    {{
                      selectedCustomer
                        ? selectedCustomer.name
                        : customerLoading
                          ? 'Memuat customer...'
                          : 'Pilih Customer'
                    }}
                  </span>

                  <span class="fas-select-chevron">
                    <FasIcon name="chevron-down" />
                  </span>
                </button>

                <div
                  v-if="customerDropdownOpen"
                  class="fas-dropdown"
                >
                  <div class="fas-dropdown-search">
                    <input
                      v-model="customerSearch"
                      type="text"
                      class="field-input"
                      :disabled="customerCreating"
                      :placeholder="
                        customerCreating
                          ? 'Mendaftarkan customer...'
                          : 'Cari customer atau ketik nama baru lalu Enter...'
                      "
                      autofocus
                      @keydown.enter.prevent.stop="
                        createCustomerFromSearch
                      "
                    />
                  </div>

                  <div
                    class="fas-dropdown-list customer-dropdown-list"
                  >
                    <button
                      v-for="customer in filteredCustomers"
                      :key="customer.id"
                      type="button"
                      class="fas-dropdown-option"
                      @click="selectCustomer(customer)"
                    >
                      <span class="fas-option-main">
                        <strong>
                          {{ customer.name }}
                        </strong>

                        <small>
                          {{
                            customer.customer_code
                          }}
                          ·
                          {{
                            customer.phone || 'Tanpa telepon'
                          }}
                        </small>
                      </span>

                      <span class="fas-option-side">
                        {{
                          customer.is_pkp
                            ? 'PKP'
                            : 'NON-PKP'
                        }}
                      </span>
                    </button>

                    <button
                      v-if="
                        !customerLoading &&
                        filteredCustomers.length === 0 &&
                        canCreateCustomerFromSearch
                      "
                      type="button"
                      class="customer-enter-create"
                      :disabled="customerCreating"
                      @click="createCustomerFromSearch"
                    >
                      <span class="customer-enter-icon">
                        <FasIcon name="enter" />
                      </span>

                      <span>
                        <strong>
                          Tekan Enter untuk daftarkan
                        </strong>

                        <small>
                          “{{ customerSearch.trim() }}”
                        </small>
                      </span>
                    </button>

                    <div
                      v-else-if="
                        !customerLoading &&
                        filteredCustomers.length === 0
                      "
                      class="fas-dropdown-empty"
                    >
                      Ketik minimal 3 karakter untuk customer baru.
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="field">
              <label class="field-label">
                Kode Customer
              </label>

              <div class="readonly-field readonly-box">
                {{
                  order.store_code || '—'
                }}
              </div>
            </div>

            <div class="field">
              <label class="field-label">
                Tanggal Pesanan
              </label>

              <input
                v-model="order.order_date"
                type="date"
                class="field-input"
              />
            </div>

            <div class="field">
              <label class="field-label">
                Nomor Telepon
              </label>

              <input
                v-model="order.phone"
                type="text"
                class="field-input"
                placeholder="Optional"
              />
            </div>

            <div class="field">
              <label class="field-label">
                Status
              </label>

              <div class="readonly-field">
                <span class="status-dot"></span>
                {{
                  getOrderStatusLabel(
                    currentOrderStatus,
                  )
                }}
              </div>
            </div>

            <div
              class="field"
              @click.stop
            >
              <label class="field-label">
                Tempo Pembayaran
              </label>

              <div class="fas-select-shell payment-terms-shell">
                <button
                  type="button"
                  :class="[
                    'fas-select-trigger',
                    'payment-terms-trigger',
                    paymentTermsDays !== null
                      ? 'payment-terms-selected'
                      : '',
                  ]"
                  @click="togglePaymentTermsDropdown"
                >
                  <span class="payment-terms-trigger-copy">
                    <span class="payment-terms-icon">
                      <FasIcon name="calendar" />
                    </span>

                    <strong>
                      {{
                        paymentTermsLabel
                      }}
                    </strong>
                  </span>

                  <span class="fas-select-chevron">
                    <FasIcon name="chevron-down" />
                  </span>
                </button>

                <div
                  v-if="paymentTermsDropdownOpen"
                  class="payment-terms-compact"
                >
                  <template
                    v-if="!manualPaymentTermsOpen"
                  >
                    <button
                      v-for="option in paymentTermsOptions"
                      :key="option.value"
                      type="button"
                      :class="[
                        'payment-terms-chip',
                        String(option.value) ===
                        paymentTermsMode
                          ? 'payment-terms-chip-active'
                          : '',
                      ]"
                      @click="
                        selectPaymentTerms(
                          option.value,
                        )
                      "
                    >
                      {{
                        option.label
                      }}
                    </button>

                    <button
                      type="button"
                      :class="[
                        'payment-terms-chip',
                        'payment-terms-chip-manual',
                        paymentTermsMode ===
                        'MANUAL'
                          ? 'payment-terms-chip-active'
                          : '',
                      ]"
                      @click="openManualPaymentTerms"
                    >
                      Manual
                    </button>
                  </template>

                  <div
                    v-else
                    class="payment-terms-manual-compact"
                  >
                    <input
                      v-model="manualPaymentTermsDays"
                      type="number"
                      min="1"
                      max="365"
                      inputmode="numeric"
                      class="payment-terms-manual-input"
                      placeholder="45"
                      autofocus
                      @keydown.enter.prevent.stop="
                        commitManualPaymentTerms
                      "
                    />

                    <span>
                      HARI
                    </span>

                    <button
                      type="button"
                      class="payment-terms-manual-enter"
                      @click="
                        commitManualPaymentTerms
                      "
                    >
                      <FasIcon name="enter" />
                    </button>

                    <button
                      type="button"
                      class="payment-terms-manual-cancel"
                      @click="
                        manualPaymentTermsOpen = false
                      "
                    >
                      <FasIcon name="close" />
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div class="field span-two">
              <label class="field-label">
                Alamat Pengiriman
              </label>

              <textarea
                v-model="order.address"
                class="field-textarea"
                rows="3"
                placeholder="Alamat toko / customer"
              ></textarea>
            </div>
          </div>
        </section>

        <section class="panel order-items-panel">
          <div class="panel-header">
            <div>
              <p class="panel-kicker">
                Produk Pesanan
              </p>

              <h2 class="panel-title">
                Produk Pesanan
              </h2>
            </div>

            <span class="item-count">
              {{ order.items.length }} item
            </span>
          </div>

          <div class="product-picker">
            <div class="search-field">
              <FasIcon name="search" class="search-icon" />

              <input
                v-model="productSearch"
                type="text"
                class="field-input search-input"
                placeholder="Cari produk, SKU, kode, barcode..."
                @focus="productDropdownOpen = true"
                @click.stop
              />
            </div>

            <div
              class="fas-select-shell"
              @click.stop
            >
              <button
                type="button"
                class="fas-select-trigger product-select-trigger"
                :disabled="loading"
                @click="toggleProductDropdown"
              >
                <span class="product-trigger-copy">
                  <strong>
                    {{
                      selectedProduct
                        ? selectedProduct.name
                        : loading
                          ? 'Memuat produk...'
                          : 'Pilih Produk'
                    }}
                  </strong>

                  <small v-if="selectedProduct">
                    {{
                      selectedProduct.product_code
                    }}
                    ·
                    Stok
                    {{
                      selectedProduct.stock
                    }}
                    {{
                      selectedProduct.unit
                    }}
                  </small>

                  <small v-else>
                    Produk yang tersedia untuk pemesanan
                  </small>
                </span>

                <span class="fas-select-chevron">
                  <FasIcon name="chevron-down" />
                </span>
              </button>

              <div
                v-if="productDropdownOpen"
                class="fas-dropdown product-dropdown"
              >
                <div class="product-dropdown-header">
                  <div>
                    <strong>
                      Pilih Produk
                    </strong>

                    <small>
                      {{
                        filteredProducts.length
                      }}
                      produk tersedia
                    </small>
                  </div>

                  <span>
                    Klik salah satu produk
                  </span>
                </div>

                <div
                  class="fas-dropdown-list product-dropdown-list"
                >
                  <button
                    v-for="product in filteredProducts"
                    :key="product.id"
                    type="button"
                    :class="[
                      'fas-dropdown-option',
                      'product-option',
                      selectedProductID === product.id
                        ? 'product-option-selected'
                        : '',
                    ]"
                    @click="selectProduct(product)"
                  >
                    <span class="product-option-copy">
                      <strong>
                        {{ product.name }}
                      </strong>

                      <small>
                        <span>
                          {{
                            product.product_code
                          }}
                        </span>

                        <span>
                          {{
                            product.sku
                          }}
                        </span>

                        <span v-if="product.barcode">
                          {{
                            product.barcode
                          }}
                        </span>
                      </small>
                    </span>

                    <span class="product-option-stock">
                      <small>
                        Stok
                      </small>

                      <strong>
                        {{
                          product.stock
                        }}
                        {{
                          product.unit
                        }}
                      </strong>
                    </span>

                    <span class="product-option-price">
                      <small>
                        Harga
                      </small>

                      <strong>
                        {{
                          formatCurrency(
                            product.price,
                          )
                        }}
                      </strong>
                    </span>
                  </button>

                  <div
                    v-if="
                      !loading &&
                      filteredProducts.length === 0
                    "
                    class="fas-dropdown-empty"
                  >
                    Produk tidak ditemukan.
                  </div>
                </div>
              </div>
            </div>

            <div class="qty-stepper add-qty-stepper">
              <button
                type="button"
                class="qty-stepper-button"
                aria-label="Kurangi jumlah"
                @click="decreaseSelectedQuantity"
              >
                <FasIcon name="minus" />
              </button>

              <input
                v-model.number="
                  selectedQuantity
                "
                type="number"
                min="1"
                step="1"
                inputmode="numeric"
                class="qty-stepper-input"
                aria-label="Jumlah produk"
                @change="
                  selectedQuantity =
                    normalizeQty(
                      selectedQuantity,
                    )
                "
              />

              <button
                type="button"
                class="qty-stepper-button"
                aria-label="Tambah jumlah"
                @click="increaseSelectedQuantity"
              >
                <FasIcon name="plus" />
              </button>
            </div>

            <button
              type="button"
              class="add-item-button"
              @click="addProduct"
            >
              <span class="add-item-icon">
                <FasIcon name="plus" />
              </span>
              Tambah
            </button>
          </div>

          <div
            v-if="selectedProduct"
            class="selected-product-preview"
          >
            <div>
              <p class="preview-name">
                {{ selectedProduct.name }}
              </p>

              <p class="preview-meta">
                {{
                  selectedProduct.product_code
                }}
                ·
                {{
                  selectedProduct.sku
                }}
                ·
                {{
                  selectedProduct.unit
                }}
              </p>
            </div>

            <div class="preview-stock">
              <span>
                Stok
              </span>

              <strong>
                {{
                  selectedProduct.stock
                }}
                {{
                  selectedProduct.unit
                }}
              </strong>
            </div>

            <div class="preview-price">
              <span>
                Harga Jual
              </span>

              <strong>
                {{
                  formatCurrency(
                    selectedProduct.price,
                  )
                }}
              </strong>
            </div>
          </div>

          <div
            v-if="loading"
            class="empty-state"
          >
            Memuat produk...
          </div>

          <div
            v-else-if="
              order.items.length === 0
            "
            class="empty-state"
          >
            Belum ada produk dalam pesanan.
          </div>

          <div
            v-else
            class="table-scroll"
          >
            <table class="order-table">
              <thead>
                <tr>
                  <th>Produk</th>
                  <th>Stok</th>
                  <th>Qty</th>
                  <th>Unit</th>
                  <th>Harga</th>
                  <th>Subtotal</th>
                  <th></th>
                </tr>
              </thead>

              <tbody>
                <tr
                  v-for="
                    item
                    in order.items
                  "
                  :key="
                    item.product_id
                  "
                >
                  <td>
                    <p class="item-name">
                      {{ item.name }}
                    </p>

                    <p class="item-code">
                      {{
                        item.product_code
                      }}
                      ·
                      {{ item.sku }}
                    </p>
                  </td>

                  <td>
                    <span
                      :class="[
                        'stock-chip',
                        Number(item.qty) >
                        Number(item.stock)
                          ? 'stock-danger'
                          : 'stock-safe',
                      ]"
                    >
                      {{
                        item.stock
                      }}
                      {{
                        item.unit
                      }}
                    </span>
                  </td>

                  <td>
                    <div class="qty-stepper table-qty-stepper">
                      <button
                        type="button"
                        class="qty-stepper-button"
                        aria-label="Kurangi quantity"
                        @click="
                          decreaseItemQty(item)
                        "
                      >
                        <FasIcon name="minus" />
                      </button>

                      <input
                        v-model.number="
                          item.qty
                        "
                        type="number"
                        min="1"
                        step="1"
                        inputmode="numeric"
                        class="qty-stepper-input"
                        aria-label="Quantity item"
                        @change="
                          updateQty(item)
                        "
                      />

                      <button
                        type="button"
                        class="qty-stepper-button"
                        aria-label="Tambah quantity"
                        @click="
                          increaseItemQty(item)
                        "
                      >
                        <FasIcon name="plus" />
                      </button>
                    </div>

                    <p
                      v-if="
                        Number(item.qty) >
                        Number(item.stock)
                      "
                      class="stock-warning"
                    >
                      Melebihi stok
                    </p>
                  </td>

                  <td>
                    {{ item.unit }}
                  </td>

                  <td class="money-cell">
                    {{
                      formatCurrency(
                        item.price,
                      )
                    }}
                  </td>

                  <td class="money-cell strong-money">
                    {{
                      formatCurrency(
                        Number(
                          item.price,
                        ) *
                        Number(
                          item.qty,
                        ),
                      )
                    }}
                  </td>

                  <td>
                    <button
                      type="button"
                      class="remove-button"
                      title="Hapus dari pesanan"
                      @click="
                        removeItem(
                          item.product_id,
                        )
                      "
                    >
                      <FasIcon name="close" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="order-total">
            <div>
              <p>
                Total Qty
              </p>

              <strong>
                {{
                  formatNumber(
                    totalQty,
                  )
                }}
              </strong>
            </div>

            <div class="grand-total">
              <p>
                Nilai Pesanan
              </p>

              <strong>
                {{
                  formatCurrency(
                    subtotal,
                  )
                }}
              </strong>
            </div>
          </div>
        </section>

        <section class="panel">
          <div class="panel-header">
            <div>
              <p class="panel-kicker">
                Additional Information
              </p>

              <h2 class="panel-title">
                Catatan Pesanan
              </h2>
            </div>
          </div>

          <textarea
            v-model="order.notes"
            class="field-textarea"
            rows="4"
            placeholder="Tambahkan catatan khusus untuk gudang, pengiriman, atau pelanggan..."
          ></textarea>
        </section>
      </main>

      <Teleport to="body" :disabled="!compactWorkspace">
      <div
        v-if="sidePanelsVisible && compactWorkspace"
        class="side-panel-backdrop"
        aria-hidden="true"
        @click="closeSidePanel"
      ></div>

      <aside
        v-if="sidePanelsVisible"
        id="operations-side-panel"
        :class="['side-column', { 'side-panel-drawer': compactWorkspace }]"
        :role="compactWorkspace ? 'dialog' : 'complementary'"
        :aria-modal="compactWorkspace ? true : undefined"
        aria-label="Panel analisis dan alur pesanan"
      >
        <header v-if="compactWorkspace" class="side-drawer-header">
          <strong>Analisis Pesanan</strong>
          <button type="button" class="panel-close-button" aria-label="Tutup panel pesanan" @click="closeSidePanel">
            <FasIcon name="close" />
          </button>
        </header>

        <section
          v-if="showDSSPanel"
          class="panel dss-panel"
        >
          <div class="panel-header">
            <div>
              <p class="panel-kicker">
                Analisis Distribusi
              </p>

              <h2 class="panel-title">
                Analisis Prioritas
              </h2>
            </div>

            <div class="panel-header-actions">
              <span
                :class="[
                  'dss-badge',
                  dssResult.className,
                ]"
              >
                {{
                  dssResult.shortLabel
                }}
              </span>

              <button
                type="button"
                class="panel-close-button"
                title="Sembunyikan Analisis Prioritas"
                @click="
                  showDSSPanel = false
                "
              >
                <FasIcon name="close" />
              </button>
            </div>
          </div>

          <div class="dss-input-grid">
            <div class="dss-input-card">
              <span>
                Nilai Pesanan
              </span>

              <strong>
                {{
                  formatCurrency(
                    subtotal,
                  )
                }}
              </strong>
            </div>

            <div class="dss-input-card">
              <span>
                Total Qty
              </span>

              <strong>
                {{
                  formatNumber(
                    totalQty,
                  )
                }}
              </strong>
            </div>
          </div>

          <div class="dss-result-box">
            <p class="dss-result-label">
              Klasifikasi
            </p>

            <h3>
              {{
                dssResult.cluster
              }}
            </h3>
          </div>

          <div class="priority-box">
            <p>
              Prioritas Distribusi
            </p>

            <strong>
              {{
                dssResult.priority
              }}
            </strong>
          </div>

          <div class="recommendation-box">
            <p class="recommendation-label">
              Rekomendasi
            </p>

            <p class="recommendation-text">
              {{
                dssResult.recommendation
              }}
            </p>
          </div>

          <div class="dss-note">
            Prioritas distribusi ditentukan berdasarkan nilai transaksi,
            jumlah barang, dan kebutuhan penanganan pesanan.
            Pesanan bervolume besar akan diprioritaskan sesuai kapasitas
            gudang dan rencana pengiriman.
          </div>
        </section>

        <section
          v-if="showWorkflowPanel"
          class="panel workflow-panel"
        >
          <div class="panel-header">
            <div>
              <p class="panel-kicker">
                Proses Pesanan
              </p>

              <h2 class="panel-title">
                Alur Pesanan
              </h2>
            </div>

            <button
              type="button"
              class="panel-close-button"
              title="Sembunyikan Alur Pesanan"
              @click="
                showWorkflowPanel = false
              "
            >
              <FasIcon name="close" />
            </button>
          </div>

          <div class="workflow-list">
            <div
              :class="[
                'workflow-item',
                workflowItemClass('DRAFT'),
              ]"
            >
              <span class="workflow-number">
                1
              </span>

              <div>
                <strong>
                  Draft
                </strong>

                <p>
                  Input customer dan produk
                </p>
              </div>
            </div>

            <div class="workflow-line"></div>

            <div
              :class="[
                'workflow-item',
                workflowItemClass('CONFIRMED'),
              ]"
            >
              <span class="workflow-number">
                2
              </span>

              <div>
                <strong>
                  Dikonfirmasi
                </strong>

                <p>
                  Pesanan telah dikonfirmasi
                </p>
              </div>
            </div>

            <div class="workflow-line"></div>

            <div
              :class="[
                'workflow-item',
                workflowItemClass('PICKING'),
              ]"
            >
              <span class="workflow-number">
                3
              </span>

              <div>
                <strong>
                  Pengambilan
                </strong>

                <p>
                  Barang sedang disiapkan
                </p>
              </div>
            </div>

            <div class="workflow-line"></div>

            <div
              :class="[
                'workflow-item',
                workflowItemClass('PACKED'),
              ]"
            >
              <span class="workflow-number">
                4
              </span>

              <div>
                <strong>
                  Dikemas
                </strong>

                <p>
                  Barang siap dikirim
                </p>
              </div>
            </div>

            <div class="workflow-line"></div>

            <div
              :class="[
                'workflow-item',
                workflowItemClass('SHIPPED'),
              ]"
            >
              <span class="workflow-number">
                5
              </span>

              <div>
                <strong>
                  Dikirim
                </strong>

                <p>
                  Pengiriman telah dicatat
                </p>
              </div>
            </div>
          </div>
        </section>
      </aside>
      </Teleport>
    </div>
  </div>
</template>

<style scoped>
.operations-page {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  padding: 1.25rem;
  container-type: inline-size;
}

.page-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1.5rem;
}

.eyebrow,
.panel-kicker {
  color: var(--fas-accent-text);
  font-size: 0.68rem;
  font-weight: 800;
  letter-spacing: 0.13em;
  text-transform: uppercase;
}

.page-title {
  margin-top: 0.45rem;
  color: var(--fas-text);
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.025em;
}

.page-description {
  margin-top: 0.7rem;
  max-width: 48rem;
  color: var(--fas-muted);
  font-size: 0.8rem;
  line-height: 1.4rem;
}

.header-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.7rem;
}

.view-panel-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.28rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-card);
}

.panel-toggle-button {
  min-height: 2.4rem;
  display: inline-flex;
  align-items: center;
  gap: 0.42rem;
  border: 1px solid transparent;
  border-radius: 0.68rem;
  padding: 0 0.72rem;
  color: var(--fas-muted);
  font-size: 0.64rem;
  font-weight: 800;
  transition:
    border-color 0.16s ease,
    background 0.16s ease,
    color 0.16s ease;
}

.panel-toggle-button:hover,
.panel-toggle-button:focus-visible {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  outline: none;
}

.panel-toggle-active {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.panel-toggle-dot {
  width: 0.38rem;
  height: 0.38rem;
  border-radius: 999px;
  background: currentColor;
}

.side-panel-icon {
  width: 1.05rem;
  height: 1.05rem;
  display: inline-grid;
  place-items: center;
  color: currentColor;
}

.side-panel-icon svg {
  width: 100%;
  height: 100%;
  stroke: currentColor;
  stroke-width: 1.6;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.panel-header-actions {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.panel-close-button {
  width: 1.75rem;
  height: 1.75rem;
  display: grid;
  place-items: center;
  border: 1px solid var(--fas-border);
  border-radius: 999px;
  background: rgba(15, 23, 42, 0.82);
  color: var(--fas-muted);
  font-size: 0.9rem;
  font-weight: 800;
  line-height: 1;
  transition:
    border-color 0.16s ease,
    background 0.16s ease,
    color 0.16s ease;
}

.panel-close-button:hover,
.panel-close-button:focus-visible {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  outline: none;
}

.primary-button,
.secondary-button {
  min-height: 3rem;
  border-radius: 0.85rem;
  padding: 0 1rem;
  font-size: 0.72rem;
  font-weight: 700;
}

.primary-button {
  background: var(--fas-accent);
  color: white;
}

.primary-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.workflow-action-button,
.workflow-ready-button {
  min-height: 3rem;
  border-radius: 0.85rem;
  padding: 0 1rem;
  font-size: 0.68rem;
  font-weight: 800;
  white-space: nowrap;
}

.workflow-action-button {
  border: 1px solid rgba(52, 211, 153, 0.26);
  background: rgba(16, 185, 129, 0.14);
  color: #34d399;
}

.workflow-action-button:hover:not(:disabled) {
  border-color: rgba(52, 211, 153, 0.42);
  background: rgba(16, 185, 129, 0.22);
}

.workflow-action-button:disabled,
.workflow-ready-button:disabled {
  cursor: not-allowed;
  opacity: 0.62;
}

.workflow-ready-button {
  border: 1px solid rgba(96, 165, 250, 0.28);
  background: rgba(37, 99, 235, 0.14);
  color: #60a5fa;
}

.secondary-button {
  border: 1px solid var(--fas-border);
  background: var(--fas-card);
  color: var(--fas-text-soft);
}

.message {
  margin-top: 1.2rem;
  border-radius: 0.9rem;
  padding: 0.85rem 1rem;
  font-size: 0.72rem;
}

.error-message {
  border: 1px solid rgb(248 113 113 / 0.2);
  background: rgb(248 113 113 / 0.08);
  color: #f87171;
}

.success-message {
  border: 1px solid rgb(52 211 153 / 0.2);
  background: rgb(52 211 153 / 0.08);
  color: #34d399;
}

.summary-grid {
  margin-top: 1.6rem;
  display: grid;
  grid-template-columns:
    repeat(4, minmax(0, 1fr));
  gap: 1rem;
}

.summary-card,
.panel {
  border: 1px solid var(--fas-border);
  background: var(--fas-card);
  box-shadow: 0 18px 45px var(--fas-shadow);
}

.summary-card {
  border-radius: 1.2rem;
  padding: 1.2rem;
}

.summary-label {
  color: var(--fas-muted);
  font-size: 0.68rem;
}

.summary-value {
  margin-top: 0.55rem;
  color: var(--fas-text);
  font-size: 1.6rem;
  font-weight: 800;
}

.summary-money {
  font-size: 1.15rem;
}

.summary-priority {
  font-size: 1.05rem;
}

.summary-helper {
  margin-top: 0.55rem;
  color: var(--fas-muted-soft);
  font-size: 0.6rem;
}

.workspace-grid {
  margin-top: 1.5rem;
  width: 100%;
  min-width: 0;
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    minmax(310px, 360px);
  gap: 1.25rem;
  align-items: start;
  transition:
    grid-template-columns 0.22s ease,
    gap 0.22s ease;
}

.workspace-grid.workspace-full {
  grid-template-columns: minmax(0, 1fr);
  gap: 0;
}

.workspace-full .main-column {
  width: 100%;
  max-width: none;
}

.workspace-full .panel {
  width: 100%;
}

.main-column,
.side-column {
  width: 100%;
  min-width: 0;
  max-width: 100%;
  display: grid;
  gap: 1.2rem;
  align-content: start;
}

.side-column {
  position: sticky;
  top: 7rem;
  z-index: 20;
  align-self: start;
  max-height: calc(100dvh - 8rem);
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: thin;
  scrollbar-color: var(--fas-border) transparent;
}

.side-panel-backdrop {
  display: none;
}

.panel {
  min-width: 0;
  max-width: 100%;
  border-radius: 1.3rem;
  overflow: hidden;
}

.order-items-panel {
  position: relative;
  z-index: 10;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  contain: inline-size;
  overflow: visible;
}

.order-items-panel .product-picker {
  position: relative;
  z-index: 40;
}

.order-items-panel .product-dropdown {
  z-index: 200;
}

.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--fas-border);
  padding: 1.15rem 1.25rem;
}

.panel-title {
  margin-top: 0.35rem;
  color: var(--fas-text);
  font-size: 1rem;
  font-weight: 800;
}

.draft-badge,
.item-count,
.dss-badge {
  border-radius: 999px;
  padding: 0.32rem 0.65rem;
  font-size: 0.55rem;
  font-weight: 800;
}

.draft-badge {
  background: rgb(148 163 184 / 0.1);
  color: var(--fas-muted);
}

.draft-badge-progress {
  background: rgba(37, 99, 235, 0.14);
  color: #60a5fa;
}

.item-count {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.form-grid {
  display: grid;
  grid-template-columns:
    repeat(2, minmax(0, 1fr));
  gap: 1rem;
  padding: 1.25rem;
}

.span-two {
  grid-column: span 2;
}

.field-label {
  display: block;
  margin-bottom: 0.45rem;
  color: var(--fas-text-soft);
  font-size: 0.68rem;
  font-weight: 700;
}

.field-input,
.field-textarea,
.readonly-field {
  width: 100%;
  border: 1px solid var(--fas-border);
  border-radius: 0.85rem;
  background: var(--fas-overlay);
  color: var(--fas-text);
  outline: none;
}

.field-input,
.readonly-field {
  min-height: 3rem;
}

.field-input {
  padding: 0 0.9rem;
  font-size: 0.75rem;
}

.field-textarea {
  resize: vertical;
  padding: 0.85rem 0.9rem;
  font-size: 0.75rem;
}

.field-input:focus,
.field-textarea:focus {
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.field-input::placeholder,
.field-textarea::placeholder {
  color: var(--fas-muted);
  opacity: 0.82;
}

.readonly-field {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0 0.9rem;
  color: var(--fas-muted);
  font-size: 0.72rem;
}

.status-dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 999px;
  background: #94a3b8;
}

.payment-terms-shell {
  position: relative;
  width: 100%;
}

.payment-terms-trigger {
  min-height: 3rem;
}

.payment-terms-selected {
  border-color: var(--fas-accent-border);
}

.payment-terms-trigger-copy {
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
}

.payment-terms-trigger-copy strong {
  overflow: hidden;
  color: var(--fas-text);
  font-size: 0.7rem;
  font-weight: 780;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.payment-terms-icon {
  color: var(--fas-accent-text);
  font-size: 0.72rem;
}

.payment-terms-compact {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.4rem;
  margin-top: 0.45rem;
}

.payment-terms-chip {
  min-height: 2.15rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--fas-border);
  border-radius: 999px;
  background: var(--fas-overlay);
  padding: 0 0.72rem;
  color: var(--fas-muted);
  font-size: 0.58rem;
  font-weight: 800;
  white-space: nowrap;
  transition:
    border-color 0.16s ease,
    background 0.16s ease,
    color 0.16s ease;
}

.payment-terms-chip:hover,
.payment-terms-chip:focus-visible {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  outline: none;
}

.payment-terms-chip-active {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.payment-terms-chip-manual {
  padding-right: 0.85rem;
  padding-left: 0.85rem;
}

.payment-terms-manual-compact {
  display: inline-grid;
  grid-template-columns:
    4.5rem
    auto
    2rem
    2rem;
  align-items: center;
  gap: 0.38rem;
  min-height: 2.2rem;
  border: 1px solid var(--fas-accent-border);
  border-radius: 999px;
  background: var(--fas-overlay);
  padding: 0.22rem 0.3rem 0.22rem 0.4rem;
}

.payment-terms-manual-input {
  width: 4.5rem;
  min-width: 0;
  height: 1.8rem;
  border: 0;
  border-radius: 999px;
  background: var(--fas-overlay);
  padding: 0 0.55rem;
  color: var(--fas-text);
  text-align: center;
  font-size: 0.62rem;
  font-weight: 800;
  outline: none;
  appearance: textfield;
  -moz-appearance: textfield;
}

.payment-terms-manual-input::-webkit-inner-spin-button,
.payment-terms-manual-input::-webkit-outer-spin-button {
  margin: 0;
  -webkit-appearance: none;
  appearance: none;
}

.payment-terms-manual-compact > span {
  color: var(--fas-muted);
  font-size: 0.54rem;
  font-weight: 800;
}

.payment-terms-manual-enter,
.payment-terms-manual-cancel {
  width: 1.8rem;
  height: 1.8rem;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 999px;
  font-size: 0.7rem;
  font-weight: 900;
}

.payment-terms-manual-enter {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.payment-terms-manual-cancel {
  background: var(--fas-muted-soft);
  color: var(--fas-muted);
}

.payment-terms-manual-enter:hover,
.payment-terms-manual-enter:focus-visible,
.payment-terms-manual-cancel:hover,
.payment-terms-manual-cancel:focus-visible {
  background: rgba(37, 99, 235, 0.34);
  color: #dbeafe;
  outline: none;
}

.product-picker {
  position: relative;
  width: 100%;
  min-width: 0;
  max-width: 100%;
  display: grid;
  grid-template-columns:
    minmax(0, 0.9fr)
    minmax(0, 1.25fr)
    auto
    112px;
  gap: 0.7rem;
  padding: 1.1rem 1.25rem;
  border-bottom: 1px solid var(--fas-border);
}

.product-picker > * {
  min-width: 0;
}

.search-field {
  position: relative;
}

.search-icon {
  position: absolute;
  left: 0.9rem;
  top: 50%;
  width: 0.95rem;
  transform: translateY(-50%);
  color: var(--fas-muted);
}

.search-input {
  padding-left: 2.7rem;
}

.product-select {
  appearance: none;
}

.fas-select-shell {
  position: relative;
  min-width: 0;
}

.fas-select-trigger {
  width: 100%;
  min-height: 3rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.85rem;
  background: var(--fas-overlay);
  padding: 0 0.9rem;
  color: var(--fas-text);
  text-align: left;
  font-size: 0.72rem;
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    background 0.18s ease;
}

.fas-select-trigger:hover,
.fas-select-trigger:focus-visible {
  border-color: var(--fas-accent-border);
  background: var(--fas-card);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
  outline: none;
}

.fas-select-trigger:disabled {
  cursor: not-allowed;
  opacity: 0.58;
}

.fas-select-chevron {
  flex: 0 0 auto;
  color: var(--fas-muted);
  font-size: 0.72rem;
}

.fas-dropdown {
  position: absolute;
  z-index: 80;
  top: calc(100% + 0.45rem);
  left: 0;
  right: 0;
  overflow: hidden;
  border: 1px solid var(--fas-accent-border);
  border-radius: 0.9rem;
  background: var(--fas-card);
  box-shadow:
    0 22px 60px rgba(0, 0, 0, 0.42),
    0 0 0 1px var(--fas-border);
  color: var(--fas-text);
}

.product-picker .fas-select-shell {
  position: static;
}

.product-dropdown {
  top: calc(100% - 0.35rem);
  left: 1.25rem;
  right: 1.25rem;
  width: auto;
  min-width: 0;
  max-width: calc(100% - 2.5rem);
  max-height: min(18rem, 48vh);
  display: flex;
  flex-direction: column;
  border-color: var(--fas-accent-border);
  border-radius: 1rem;
  background: var(--fas-card);
  box-shadow:
    0 26px 70px rgba(0, 0, 0, 0.48),
    0 0 0 1px var(--fas-border);
}

.product-dropdown-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 0.85rem 1rem;
  border-bottom: 1px solid rgba(148, 163, 184, 0.13);
  background: var(--fas-overlay);
}

.product-dropdown-header > div {
  display: flex;
  align-items: baseline;
  gap: 0.65rem;
}

.product-dropdown-header strong {
  color: var(--fas-text);
  font-size: 0.78rem;
  font-weight: 800;
}

.product-dropdown-header small,
.product-dropdown-header > span {
  color: var(--fas-muted);
  font-size: 0.58rem;
}

.product-dropdown-list {
  height: min(13.5rem, 38vh);
  max-height: min(13.5rem, 38vh);
  overflow-y: scroll;
  overscroll-behavior-y: contain;
  padding: 0.55rem;
}


.product-dropdown-list::-webkit-scrollbar {
  width: 0.52rem;
}

.product-dropdown-list::-webkit-scrollbar-track {
  margin: 0.35rem 0;
  border-radius: 999px;
  background: var(--fas-overlay);
}

.product-dropdown-list::-webkit-scrollbar-thumb {
  min-height: 2.5rem;
  border: 2px solid var(--fas-border);
  border-radius: 999px;
  background: var(--fas-muted-soft);
}

.product-dropdown-list::-webkit-scrollbar-thumb:hover {
  background: var(--fas-accent-soft);
}

.product-select-trigger {
  min-height: 3rem;
  padding-top: 0.48rem;
  padding-bottom: 0.48rem;
}

.product-trigger-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.16rem;
}

.product-trigger-copy strong {
  overflow: hidden;
  color: var(--fas-text);
  font-size: 0.7rem;
  font-weight: 760;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-trigger-copy small {
  overflow: hidden;
  color: var(--fas-muted);
  font-size: 0.56rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-option {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    minmax(105px, 0.24fr)
    minmax(125px, 0.3fr);
  align-items: center;
  gap: 1rem;
  min-height: 4.25rem;
  margin-bottom: 0.28rem;
  border: 1px solid transparent;
  padding: 0.78rem 0.9rem;
}

.product-option:last-child {
  margin-bottom: 0;
}

.product-option:hover,
.product-option:focus-visible {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
}

.product-option-selected {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
}

.product-option-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.32rem;
}

.product-option-copy > strong {
  overflow: hidden;
  color: var(--fas-text);
  font-size: 0.73rem;
  font-weight: 800;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.product-option-copy > small {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.product-option-copy > small > span {
  border: 1px solid var(--fas-border);
  border-radius: 0.35rem;
  background: var(--fas-overlay);
  padding: 0.18rem 0.38rem;
  color: var(--fas-muted);
  font-size: 0.54rem;
}

.product-option-stock,
.product-option-price {
  display: flex;
  flex-direction: column;
  gap: 0.18rem;
}

.product-option-stock {
  align-items: flex-start;
}

.product-option-price {
  align-items: flex-end;
  text-align: right;
}

.product-option-stock small,
.product-option-price small {
  color: var(--fas-muted);
  font-size: 0.52rem;
  text-transform: uppercase;
  letter-spacing: 0.08em;
}

.product-option-stock strong {
  color: var(--fas-text-soft);
  font-size: 0.67rem;
  font-weight: 760;
}

.product-option-price strong {
  color: var(--fas-accent-text);
  font-size: 0.7rem;
  font-weight: 820;
}

.fas-dropdown-search {
  padding: 0.7rem;
  border-bottom: 1px solid var(--fas-border);
  background: var(--fas-overlay);
}

.fas-dropdown-list {
  max-height: 19rem;
  overflow-x: hidden;
  overflow-y: auto;
  overscroll-behavior-y: contain;
  -webkit-overflow-scrolling: touch;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: var(--fas-border) transparent;
  padding: 0.4rem;
}

.fas-dropdown-list::-webkit-scrollbar {
  width: 0.48rem;
}

.fas-dropdown-list::-webkit-scrollbar-track {
  border-radius: 999px;
  background: transparent;
}

.fas-dropdown-list::-webkit-scrollbar-thumb {
  border: 2px solid var(--fas-card);
  border-radius: 999px;
  background: var(--fas-border);
}

.fas-dropdown-list::-webkit-scrollbar-thumb:hover {
  background: var(--fas-accent);
}

.customer-dropdown-list {
  max-height: 18rem;
}

.product-dropdown-list {
  height: min(13.5rem, 38vh);
  max-height: min(13.5rem, 38vh);
  overflow-y: scroll;
  overscroll-behavior-y: contain;
  scroll-behavior: smooth;
}

.fas-dropdown-option {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-radius: 0.68rem;
  padding: 0.75rem 0.8rem;
  color: var(--fas-text);
  text-align: left;
  transition:
    background 0.16s ease,
    color 0.16s ease;
}

.fas-dropdown-option:hover,
.fas-dropdown-option:focus-visible {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  outline: none;
}

.fas-option-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
}

.fas-option-main strong {
  overflow: hidden;
  color: var(--fas-text);
  font-size: 0.72rem;
  font-weight: 760;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fas-option-main small {
  overflow: hidden;
  color: var(--fas-muted);
  font-size: 0.58rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fas-option-side {
  flex: 0 0 auto;
  border: 1px solid var(--fas-accent-border);
  border-radius: 999px;
  background: var(--fas-accent-soft);
  padding: 0.28rem 0.5rem;
  color: var(--fas-accent-text);
  font-size: 0.56rem;
  font-weight: 800;
  letter-spacing: 0.04em;
}

.product-price {
  border: 0;
  background: transparent;
  color: var(--fas-text-soft);
  font-size: 0.66rem;
  letter-spacing: 0;
}

.fas-dropdown-list {
  background: var(--fas-card);
}


.customer-enter-create {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.65rem;
  border: 1px dashed var(--fas-accent-border);
  border-radius: 0.7rem;
  background: var(--fas-accent-soft);
  padding: 0.72rem 0.75rem;
  color: var(--fas-accent-text);
  text-align: left;
  transition:
    border-color 0.16s ease,
    background 0.16s ease;
}

.customer-enter-create:hover:not(:disabled),
.customer-enter-create:focus-visible {
  border-color: rgba(96, 165, 250, 0.5);
  background: rgba(37, 99, 235, 0.15);
  outline: none;
}

.customer-enter-create:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.customer-enter-icon {
  flex: 0 0 auto;
  width: 1.7rem;
  height: 1.7rem;
  display: grid;
  place-items: center;
  border-radius: 999px;
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  font-size: 0.72rem;
  font-weight: 900;
}

.customer-enter-create > span:last-child {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 0.14rem;
}

.customer-enter-create strong {
  color: var(--fas-text);
  font-size: 0.62rem;
  font-weight: 820;
}

.customer-enter-create small {
  overflow: hidden;
  color: var(--fas-muted);
  font-size: 0.55rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.fas-dropdown-empty {
  padding: 1.1rem 0.8rem;
  color: var(--fas-muted);
  text-align: center;
  font-size: 0.65rem;
}

.readonly-box {
  min-height: 3rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.85rem;
  background: var(--fas-overlay);
}

.qty-stepper {
  display: inline-grid;
  grid-template-columns:
    1.75rem
    minmax(0, 1fr)
    1.75rem;
  align-items: center;
  gap: 0.2rem;
  overflow: hidden;
  min-height: 2.6rem;
  border: 1px solid var(--fas-border-strong);
  border-radius: 999px;
  background: var(--fas-input);
  padding: 0.25rem;
  color: var(--fas-text);
  transition:
    border-color 0.18s ease,
    box-shadow 0.18s ease,
    background 0.18s ease;
}

.qty-stepper:focus-within {
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.qty-stepper-button {
  display: grid;
  place-items: center;
  width: 1.75rem;
  height: 1.75rem;
  margin: 0;
  border: 0;
  border-radius: 999px;
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  font-size: 0.82rem;
  font-weight: 900;
  line-height: 1;
  cursor: pointer;
  transition:
    background 0.16s ease,
    color 0.16s ease;
}

.qty-stepper-button:hover {
  background: var(--fas-accent);
  color: #ffffff;
}

.qty-stepper-button:focus-visible {
  outline: 2px solid var(--fas-accent);
  outline-offset: 1px;
}

.qty-stepper-input {
  width: 100%;
  min-width: 0;
  border: 0;
  background: transparent;
  padding: 0;
  color: var(--fas-text);
  text-align: center;
  font-size: 0.8rem;
  font-weight: 800;
  outline: none;
  appearance: textfield;
  -moz-appearance: textfield;
}

.qty-stepper-input::-webkit-inner-spin-button,
.qty-stepper-input::-webkit-outer-spin-button {
  margin: 0;
  -webkit-appearance: none;
  appearance: none;
}

.add-qty-stepper {
  width: 6.75rem;
  justify-self: center;
}

.table-qty-stepper {
  width: 6.75rem;
  min-height: 2.45rem;
}

.add-item-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.32rem;
  padding: 0 0.85rem;
  white-space: nowrap;
}

.add-item-icon {
  display: inline-grid;
  place-items: center;
  width: 0.78rem;
  height: 0.78rem;
  border-radius: 999px;
  color: inherit;
  font-size: 0.72rem;
  font-weight: 900;
  line-height: 1;
}

.qty-input {
  text-align: center;
}

.add-item-button {
  min-height: 2.65rem;
  border-radius: 0.78rem;
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  font-size: 0.68rem;
  font-weight: 800;
}

.selected-product-preview {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    auto
    auto;
  gap: 1.2rem;
  align-items: center;
  margin: 1rem 1.25rem 0;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  padding: 0.9rem 1rem;
}

.preview-name {
  color: var(--fas-text);
  font-size: 0.75rem;
  font-weight: 700;
}

.preview-meta {
  margin-top: 0.2rem;
  color: var(--fas-muted);
  font-size: 0.6rem;
}

.preview-stock,
.preview-price {
  text-align: right;
}

.preview-stock span,
.preview-price span {
  display: block;
  color: var(--fas-muted);
  font-size: 0.56rem;
}

.preview-stock strong,
.preview-price strong {
  display: block;
  margin-top: 0.15rem;
  color: var(--fas-text);
  font-size: 0.7rem;
}

.empty-state {
  display: flex;
  min-height: 12rem;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  color: var(--fas-muted);
  font-size: 0.72rem;
}

.table-scroll {
  width: 100%;
  min-width: 0;
  max-width: 100%;
  overflow-x: auto;
  overflow-y: hidden;
  overscroll-behavior-x: contain;
  scrollbar-gutter: stable;
  scrollbar-width: thin;
  scrollbar-color: var(--fas-muted-soft) transparent;
}

.table-scroll::-webkit-scrollbar {
  height: 0.48rem;
}

.table-scroll::-webkit-scrollbar-track {
  border-radius: 999px;
  background: var(--fas-overlay);
}

.table-scroll::-webkit-scrollbar-thumb {
  border: 1px solid var(--fas-border);
  border-radius: 999px;
  background: var(--fas-muted-soft);
}

.table-scroll::-webkit-scrollbar-thumb:hover {
  background: var(--fas-accent-soft);
}

.order-table {
  width: max(100%, 900px);
  min-width: 900px;
  border-collapse: separate;
  border-spacing: 0;
}

.order-table th,
.order-table td {
  border-bottom: 1px solid var(--fas-border);
  padding: 0.85rem 1rem;
  text-align: left;
}

.order-table th {
  color: var(--fas-muted);
  font-size: 0.58rem;
  font-weight: 700;
  text-transform: uppercase;
}

.order-table td {
  color: var(--fas-text-soft);
  font-size: 0.7rem;
}

.item-name {
  color: var(--fas-text);
  font-weight: 700;
}

.item-code {
  margin-top: 0.2rem;
  color: var(--fas-muted);
  font-size: 0.58rem;
}

.stock-chip {
  display: inline-flex;
  border-radius: 999px;
  padding: 0.3rem 0.6rem;
  font-size: 0.56rem;
  font-weight: 800;
}

.stock-safe {
  background: rgb(52 211 153 / 0.08);
  color: #34d399;
}

.stock-danger {
  background: rgb(248 113 113 / 0.08);
  color: #f87171;
}

.table-qty {
  width: 80px;
  min-height: 2.4rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.7rem;
  background: var(--fas-overlay);
  padding: 0 0.6rem;
  color: var(--fas-text);
  text-align: center;
}

.stock-warning {
  margin-top: 0.25rem;
  color: #f87171;
  font-size: 0.52rem;
}

.money-cell {
  text-align: right !important;
}

.strong-money {
  color: var(--fas-text) !important;
  font-weight: 800;
}

.remove-button {
  display: inline-flex;
  width: 2.2rem;
  height: 2.2rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.65rem;
  background: rgb(248 113 113 / 0.07);
  color: #f87171;
  font-size: 1rem;
}

.order-total {
  display: flex;
  justify-content: flex-end;
  gap: 2rem;
  padding: 1rem 1.25rem 1.2rem;
}

.order-total p {
  color: var(--fas-muted);
  font-size: 0.6rem;
}

.order-total strong {
  display: block;
  margin-top: 0.2rem;
  color: var(--fas-text);
  font-size: 0.9rem;
}

.grand-total {
  min-width: 190px;
  text-align: right;
}

.grand-total strong {
  color: var(--fas-accent-text);
  font-size: 1.05rem;
}

.main-column > .panel:last-child .field-textarea {
  margin: 1.1rem 1.25rem 1.25rem;
  width: calc(100% - 2.5rem);
}

.dss-panel,
.workflow-panel {
  position: static;
  top: auto;
}

.dss-input-grid {
  display: grid;
  grid-template-columns:
    repeat(2, minmax(0, 1fr));
  gap: 0.7rem;
  padding: 1rem 1.1rem 0;
}

.dss-input-card {
  border: 1px solid var(--fas-border);
  border-radius: 0.8rem;
  background: var(--fas-overlay);
  padding: 0.8rem;
}

.dss-input-card span {
  color: var(--fas-muted);
  font-size: 0.55rem;
  font-family:
    ui-monospace,
    SFMono-Regular,
    Menlo,
    monospace;
}

.dss-input-card strong {
  display: block;
  margin-top: 0.3rem;
  color: var(--fas-text);
  font-size: 0.72rem;
}

.dss-result-box,
.priority-box,
.recommendation-box {
  margin: 0.8rem 1.1rem 0;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  padding: 0.9rem;
}

.dss-result-box {
  background: var(--fas-overlay);
}

.dss-result-label,
.priority-box p,
.recommendation-label {
  color: var(--fas-muted);
  font-size: 0.58rem;
}

.dss-result-box h3 {
  margin-top: 0.35rem;
  color: var(--fas-text);
  font-size: 0.82rem;
  font-weight: 800;
}

.priority-box {
  background: var(--fas-accent-soft);
}

.priority-box strong {
  display: block;
  margin-top: 0.3rem;
  color: var(--fas-accent-text);
  font-size: 0.85rem;
  font-weight: 800;
}

.recommendation-box {
  background: var(--fas-overlay);
}

.recommendation-text {
  margin-top: 0.35rem;
  color: var(--fas-text-soft);
  font-size: 0.68rem;
  line-height: 1.15rem;
}

.dss-note {
  margin: 0.9rem 1.1rem 1.1rem;
  color: var(--fas-muted);
  font-size: 0.58rem;
  line-height: 1rem;
}

.dss-neutral {
  background: rgb(148 163 184 / 0.1);
  color: var(--fas-muted);
}

.dss-retail {
  background: rgb(59 130 246 / 0.1);
  color: #60a5fa;
}

.dss-regular {
  background: rgb(52 211 153 / 0.1);
  color: #34d399;
}

.dss-high-volume {
  border-color: rgba(245, 158, 11, 0.34);
  background:
    linear-gradient(
      180deg,
      rgba(120, 53, 15, 0.22),
      rgba(69, 26, 3, 0.14)
    );
  color: #fbbf24;
}

.dss-high-volume .priority-value,
.dss-high-volume .dss-value,
.dss-high-volume strong {
  color: #fbbf24;
}


.dss-vvip {
  background: rgb(168 85 247 / 0.12);
  color: #c084fc;
}


.workflow-panel .panel-header {
  padding-bottom: 0.95rem;
}

.workflow-panel .workflow-list {
  padding-top: 0.85rem;
  padding-bottom: 0.95rem;
}

.workflow-panel .workflow-item {
  min-height: 2.9rem;
}

.workflow-panel .workflow-item p {
  line-height: 0.95rem;
}

.workflow-list {
  padding: 1.1rem 1.25rem 1.25rem;
}

.workflow-item {
  display: flex;
  gap: 0.8rem;
  align-items: flex-start;
}

.workflow-number {
  display: flex;
  width: 1.8rem;
  height: 1.8rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--fas-border);
  border-radius: 999px;
  color: var(--fas-muted);
  font-size: 0.58rem;
  font-weight: 800;
}

.workflow-item.active
.workflow-number {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.workflow-item.completed
.workflow-number {
  border-color: rgba(52, 211, 153, 0.28);
  background: rgba(16, 185, 129, 0.12);
  color: #34d399;
}

.workflow-item.completed strong {
  color: #86efac;
}

.workflow-item strong {
  color: var(--fas-text);
  font-size: 0.68rem;
}

.workflow-item p {
  margin-top: 0.16rem;
  color: var(--fas-muted);
  font-size: 0.56rem;
}

.workflow-line {
  width: 1px;
  height: 1rem;
  margin-left: 0.9rem;
  background: var(--fas-border);
}

@media (max-width: 1180px) {
  .workspace-grid {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(270px, 32%);
    gap: 1rem;
  }

  .dss-panel {
    position: static;
  }
}

@media (max-width: 900px) {
  .product-dropdown {
    left: 1rem;
    right: 1rem;
  }

  .product-option {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(105px, 0.3fr);
  }

  .product-option-stock {
    display: none;
  }

  .summary-grid {
    grid-template-columns:
      repeat(2, minmax(0, 1fr));
  }

  .product-picker {
    grid-template-columns:
      1fr 1fr;
  }
}

@media (max-width: 700px) {
  .product-dropdown {
    left: 0.75rem;
    right: 0.75rem;
  }

  .product-dropdown-header {
    align-items: flex-start;
    flex-direction: column;
    gap: 0.25rem;
  }

  .product-option {
    grid-template-columns: 1fr;
    gap: 0.55rem;
  }

  .product-option-price {
    align-items: flex-start;
    text-align: left;
  }

  .operations-page {
    padding: 1rem;
  }

  .page-header {
    flex-direction: column;
    align-items: stretch;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions button {
    flex: 1;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .span-two {
    grid-column: auto;
  }

  .product-picker {
    grid-template-columns: 1fr;
  }

  .selected-product-preview {
    grid-template-columns: 1fr;
  }

  .preview-stock,
  .preview-price {
    text-align: left;
  }

  .side-column {
    grid-template-columns: 1fr;
  }

  .order-total {
    justify-content: space-between;
    gap: 1rem;
  }
}

@media (max-width: 520px) {
  .summary-grid {
    grid-template-columns: 1fr 1fr;
  }

  .summary-value {
    font-size: 1.25rem;
  }

  .summary-money {
    font-size: 0.9rem;
  }

  .order-total {
    flex-direction: column;
    align-items: flex-end;
  }
}

input[type='number'] {
  appearance: textfield;
  -moz-appearance: textfield;
}

input[type='number']::-webkit-inner-spin-button,
input[type='number']::-webkit-outer-spin-button {
  margin: 0;
  -webkit-appearance: none;
  appearance: none;
}


@container (max-width: 1120px) {
  .workspace-grid:not(.workspace-full) .product-picker {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(0, 1.15fr)
      auto
      104px;
    gap: 0.55rem;
  }

  .workspace-grid:not(.workspace-full) .add-qty-stepper {
    width: 6.35rem;
  }

  .workspace-grid:not(.workspace-full) .add-item-button {
    padding-right: 0.7rem;
    padding-left: 0.7rem;
  }


  .workspace-grid {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(280px, 320px);
    gap: 1rem;
  }

  .product-picker {
    grid-template-columns:
      minmax(170px, 0.9fr)
      minmax(210px, 1.15fr)
      auto
      106px;
  }

  .summary-grid {
    gap: 0.8rem;
  }
}

@container (max-width: 960px) {
  .workspace-grid {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(250px, 31%);
    gap: 0.8rem;
  }

  .dss-panel,
  .workflow-panel {
    height: auto;
  }

  .product-picker {
    grid-template-columns:
      minmax(0, 1fr)
      minmax(0, 1fr);
  }
}

@container (max-width: 680px) {
  .workspace-grid {
    grid-template-columns: 1fr;
  }

  .summary-grid {
    grid-template-columns:
      repeat(2, minmax(0, 1fr));
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .span-two {
    grid-column: auto;
  }

  .product-picker {
    grid-template-columns: 1fr;
  }
}


@container (max-width: 760px) {
  .header-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .view-panel-actions {
    order: 3;
    width: 100%;
  }

  .panel-toggle-button {
    width: 100%;
    justify-content: center;
  }
}




@container (max-width: 560px) {
  .payment-terms-compact {
    gap: 0.35rem;
  }

  .payment-terms-chip {
    min-height: 2rem;
    padding: 0 0.62rem;
  }
}


.side-panel-backdrop {
  position: fixed;
  z-index: 1200;
  inset: 0;
  display: block;
  background: rgba(2, 6, 23, 0.55);
  backdrop-filter: blur(3px);
}

.side-column.side-panel-drawer {
  position: fixed;
  z-index: 1210;
  top: max(0.75rem, env(safe-area-inset-top));
  right: max(0.75rem, env(safe-area-inset-right));
  bottom: max(0.75rem, env(safe-area-inset-bottom));
  width: min(24rem, calc(100vw - 1.5rem));
  max-height: none;
  grid-template-columns: 1fr;
  align-content: start;
  gap: 0.85rem;
  overflow-x: hidden;
  overflow-y: auto;
  border: 1px solid var(--fas-border-strong);
  border-radius: 1.2rem;
  background: var(--fas-card);
  padding: 0.65rem;
  box-shadow: var(--fas-shadow-card);
}

.side-drawer-header {
  position: sticky;
  top: -0.65rem;
  z-index: 3;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  background: var(--fas-card);
  padding: 0.65rem;
  color: var(--fas-text);
  font-size: 0.85rem;
}

.main-column > .panel {
  overflow: visible;
}

.panel-close-button {
  width: 2rem;
  height: 2rem;
  border-radius: 0.65rem;
  background: var(--fas-elevated);
}

.fas-select-chevron,
.payment-terms-icon,
.add-item-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.field-label {
  font-size: 0.75rem;
}

.field-input,
.field-textarea,
.fas-select-trigger {
  font-size: 0.85rem;
}

.payment-terms-manual-compact,
.quantity-stepper,
.add-qty-stepper {
  background: var(--fas-input);
  border-color: var(--fas-border-strong);
  color: var(--fas-text);
}

</style>
