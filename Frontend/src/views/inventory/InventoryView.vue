<script setup>
import { companyProfile, loadCompanyProfile } from '@/services/company'
import { buildProductCatalogHTML, writeProductCatalogPrint } from '@/utils/productCatalogPrint'
import FasSelect from '@/components/ui/FasSelect.vue'
import { getWarehouses, createWarehouse } from '@/services/warehouse'

import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
} from 'vue'

import { useAuthStore } from '@/stores/auth'

import {
  createProduct,
  deactivateProduct as deleteProduct,
  getProducts,
  updateProduct,
} from '@/services/product'

const authStore = useAuthStore()

const products = ref([])
const loading = ref(true)
const saving = ref(false)

const search = ref('')
const classificationFilter = ref('ALL')
const statusFilter = ref('ACTIVE')

const modalOpen = ref(false)
const modalMode = ref('create')
const selectedProduct = ref(null)

const confirmOpen = ref(false)
const pendingDeleteProduct = ref(null)
const deleting = ref(false)
const togglingProductID = ref(null)

const errorMessage = ref('')
const successMessage = ref('')
const formError = ref('')

const classificationMenuOpen = ref(false)
const unitMenuOpen = ref(false)
const measurementUnitMenuOpen = ref(false)
const classificationFilterMenuOpen = ref(false)
const statusFilterMenuOpen = ref(false)
const exportMenuOpen = ref(false)

const selectedUnit = ref('BOX')

const measurementValue = ref('')
const measurementUnit = ref('ML')

const skuTouched = ref(false)

const cameraInput = ref(null)
const galleryInput = ref(null)

const photoFile = ref(null)
const photoPreview = ref('')

const currentRole = computed(() => {
  return String(
    authStore.user?.role || '',
  ).toLowerCase()
})

const isAdmin = computed(() => {
  return [
    'admin',
    'owner',
    'administrator',
  ].includes(currentRole.value)
})

const isWarehouse = computed(() => {
  return [
    'pegawai',
    'warehouse',
    'gudang',
  ].includes(currentRole.value)
})

const canSeeInternalCost = computed(() => {
  return isAdmin.value
})

const canManageProducts = computed(() => {
  return (
    isAdmin.value ||
    isWarehouse.value
  )
})

const canDeleteProduct = computed(() => {
  return isAdmin.value
})

const classificationOptions = [
  'REGULER',
  'PREKURSOR',
]

const legacyClassificationFilterOptions = [
  {
    value: 'ALL',
    label: 'Semua klasifikasi',
  },
  {
    value: 'REGULER',
    label: 'REGULER',
  },
  {
    value: 'PREKURSOR',
    label: 'PREKURSOR',
  },
]

const statusFilterOptions = [
  {
    value: 'ALL',
    label: 'Semua status',
  },
  {
    value: 'ACTIVE',
    label: 'Aktif',
  },
  {
    value: 'INACTIVE',
    label: 'Nonaktif',
  },
]

const unitOptions = [
  'BOX',
  'BTL',
  'PCS',
  'LUSIN',
  'STRIP',
  'TUBE',
  'SACHET',
  'AMP',
  'VIAL',
  'PACK',
  'POT',
  'KALENG',
  'ROLL',
]

const measurementUnitOptions = [
  'ML',
  'L',
  'MCG',
  'MG',
  'G',
  'KG',
]

const quickMeasurements = [
  5,
  10,
  15,
  20,
  30,
  50,
  60,
  75,
  100,
  120,
  150,
  200,
  250,
  500,
]

const form = reactive({
  warehouse_id: '',
  sku: '',
  name: '',
  classification: 'REGULER',
  unit: 'BOX',
  purchase_price: '',
  price: '',
  stock: '',
  min_stock: 30,
  barcode: '',
})

const totalProducts = computed(() => {
  return products.value.length
})

const activeProducts = computed(() => {
  return products.value.filter(
    (product) => product.is_active,
  ).length
})

const precursorProducts = computed(() => {
  return products.value.filter(
    (product) =>
      product.classification === 'PREKURSOR' &&
      product.is_active,
  ).length
})

const lowStockProducts = computed(() => {
  return products.value.filter(
    (product) =>
      product.is_active &&
      Number(product.stock) <=
        Number(product.min_stock ?? 30),
  ).length
})

const selectedClassificationFilterLabel = computed(() => {
  return (
    classificationFilterOptions.value.find(
      (item) =>
        item.value ===
        classificationFilter.value,
    )?.label ||
    'Semua klasifikasi'
  )
})

const selectedStatusFilterLabel = computed(() => {
  return (
    statusFilterOptions.find(
      (item) =>
        item.value ===
        statusFilter.value,
    )?.label ||
    'Semua status'
  )
})

const stockUnitLabel = computed(() => {
  return selectedUnit.value || 'UNIT'
})

const stockState = (product) => {
  const stock = Number(product.stock || 0)
  const minimum = Number(product.min_stock ?? 30)

  if (stock <= 0) {
    return {
      label: 'HABIS',
      className: 'stock-status-empty',
    }
  }

  if (stock <= minimum) {
    return {
      label: 'SEGERA PESAN',
      className: 'stock-status-low',
    }
  }

  return {
    label: 'AMAN',
    className: 'stock-status-safe',
  }
}

const marginAmount = computed(() => {
  const purchase = Number(
    form.purchase_price || 0,
  )

  const selling = Number(
    form.price || 0,
  )

  return selling - purchase
})

const marginPercentage = computed(() => {
  const selling = Number(
    form.price || 0,
  )

  if (selling <= 0) {
    return 0
  }

  return (
    marginAmount.value /
    selling
  ) * 100
})

const filteredProducts = computed(() => {
  const keyword = search.value
    .trim()
    .toLowerCase()

  return products.value.filter((product) => {
    const name = String(
      product.name || '',
    ).toLowerCase()

    const sku = String(
      product.sku || '',
    ).toLowerCase()

    const productCode = String(
      product.product_code || '',
    ).toLowerCase()

    const barcode = String(
      product.barcode || '',
    ).toLowerCase()

    const matchesSearch =
      !keyword ||
      name.includes(keyword) ||
      sku.includes(keyword) ||
      productCode.includes(keyword) ||
      barcode.includes(keyword)

    const matchesClassification =
      classificationFilter.value === 'ALL' ||
      (classificationFilter.value === 'UNASSIGNED'
        ? !product.warehouse_id
        : product.warehouse_id === classificationFilter.value)

    const matchesStatus =
      statusFilter.value === 'ALL' ||
      (
        statusFilter.value === 'ACTIVE' &&
        product.is_active
      ) ||
      (
        statusFilter.value === 'INACTIVE' &&
        !product.is_active
      )

    return (
      matchesSearch &&
      matchesClassification &&
      matchesStatus
    )
  })
})

const catalogProducts = computed(() => {
  return filteredProducts.value.filter(
    (product) => product.is_active,
  )
})

const formatCurrency = (value) => {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(
    Number(value || 0),
  )
}

const normalizeMeasurement = () => {
  if (
    measurementValue.value === '' ||
    measurementValue.value === null
  ) {
    return ''
  }

  const value = Number(
    measurementValue.value,
  )

  if (
    Number.isNaN(value) ||
    value <= 0
  ) {
    return ''
  }

  const numeric =
    Number.isInteger(value)
      ? String(value)
      : String(value).replace(
          /0+$/,
          '',
        )

  const unit =
    measurementUnit.value

  if (unit === 'ML') {
    return `${numeric}ml`
  }

  if (unit === 'MCG') {
    return `${numeric}mcg`
  }

  if (unit === 'MG') {
    return `${numeric}mg`
  }

  if (unit === 'G') {
    return `${numeric}g`
  }

  if (unit === 'KG') {
    return `${numeric}kg`
  }

  if (unit === 'L') {
    return `${numeric}L`
  }

  return `${numeric}${unit}`
}

const buildProductName = () => {
  const baseName =
    form.name.trim()

  const measurement =
    normalizeMeasurement()

  if (!measurement) {
    return baseName
  }

  const normalizedName =
    baseName
      .toLowerCase()
      .replace(/\s+/g, '')

  const normalizedMeasurement =
    measurement
      .toLowerCase()
      .replace(/\s+/g, '')

  if (
    normalizedName.endsWith(
      normalizedMeasurement,
    )
  ) {
    return baseName
  }

  return `${baseName} ${measurement}`
}

const generateSKU = (name) => {
  const cleaned = name
    .toUpperCase()
    .replace(
      /[^A-Z0-9 ]/g,
      ' ',
    )
    .replace(/\s+/g, ' ')
    .trim()

  if (!cleaned) {
    return ''
  }

  const parts =
    cleaned.split(' ')

  const first =
    parts[0]?.slice(0, 8) ||
    ''

  const tail =
    parts.slice(1)

  const second =
    tail[0]?.slice(0, 8) ||
    ''

  const lastNumeric =
    [...tail]
      .reverse()
      .find((part) =>
        /\d/.test(part),
      )

  const skuParts = [
    first,
    second,
  ]

  if (
    lastNumeric &&
    lastNumeric !== second
  ) {
    skuParts.push(
      lastNumeric.slice(0, 8),
    )
  }

  return skuParts
    .filter(Boolean)
    .join('-')
    .slice(0, 40)
}

const refreshAutoSKU = () => {
  if (skuTouched.value) {
    return
  }

  form.sku =
    generateSKU(
      buildProductName(),
    )
}

const handleNameInput = () => {
  refreshAutoSKU()
}

const handleMeasurementInput = () => {
  refreshAutoSKU()
}

const handleSKUInput = () => {
  skuTouched.value =
    true
}

const regenerateSKU = () => {
  skuTouched.value =
    false

  form.sku =
    generateSKU(
      buildProductName(),
    )
}

const useQuickMeasurement = (value) => {
  measurementValue.value =
    value

  refreshAutoSKU()
}

const splitProductNameMeasurement = (name) => {
  const source = String(name || '').trim()

  const match = source.match(
    /^(.*?)\s+(\d+(?:[.,]\d+)?)\s*(ML|L|MCG|MG|G|KG)$/i,
  )

  if (!match) {
    return {
      name: source,
      value: '',
      unit: 'ML',
    }
  }

  return {
    name: match[1].trim(),
    value: match[2].replace(',', '.'),
    unit: match[3].toUpperCase(),
  }
}

const marginAmountFor = (product) => {
  const purchase = Number(
    product.purchase_price || 0,
  )

  const selling = Number(
    product.price || 0,
  )

  if (
    purchase <= 0 ||
    selling <= 0
  ) {
    return null
  }

  return selling - purchase
}

const marginPercentageFor = (product) => {
  const selling = Number(
    product.price || 0,
  )

  const margin =
    marginAmountFor(product)

  if (
    selling <= 0 ||
    margin === null
  ) {
    return null
  }

  return (
    margin /
    selling
  ) * 100
}

const closeDropdowns = () => {
  classificationMenuOpen.value =
    false

  unitMenuOpen.value =
    false

  measurementUnitMenuOpen.value =
    false

  classificationFilterMenuOpen.value =
    false

  statusFilterMenuOpen.value =
    false

  exportMenuOpen.value =
    false
}

const toggleClassificationMenu = () => {
  const next =
    !classificationMenuOpen.value

  closeDropdowns()

  classificationMenuOpen.value =
    next
}

const toggleUnitMenu = () => {
  const next =
    !unitMenuOpen.value

  closeDropdowns()

  unitMenuOpen.value =
    next
}

const toggleMeasurementUnitMenu = () => {
  const next =
    !measurementUnitMenuOpen.value

  closeDropdowns()

  measurementUnitMenuOpen.value =
    next
}

const toggleClassificationFilterMenu = () => {
  const next =
    !classificationFilterMenuOpen.value

  closeDropdowns()

  classificationFilterMenuOpen.value =
    next
}

const toggleStatusFilterMenu = () => {
  const next =
    !statusFilterMenuOpen.value

  closeDropdowns()

  statusFilterMenuOpen.value =
    next
}

const toggleExportMenu = () => {
  const next =
    !exportMenuOpen.value

  closeDropdowns()

  exportMenuOpen.value =
    next
}

const selectClassification = (
  classification,
) => {
  form.classification =
    classification

  classificationMenuOpen.value =
    false
}

const selectClassificationFilter = (
  value,
) => {
  classificationFilter.value =
    value

  classificationFilterMenuOpen.value =
    false
}

const selectStatusFilter = (
  value,
) => {
  statusFilter.value =
    value

  statusFilterMenuOpen.value =
    false
}

const selectUnit = (unit) => {
  selectedUnit.value =
    unit

  form.unit =
    unit

  unitMenuOpen.value =
    false
}

const selectMeasurementUnit = (unit) => {
  measurementUnit.value =
    unit

  measurementUnitMenuOpen.value =
    false

  refreshAutoSKU()
}

const clearMessage = () => {
  errorMessage.value = ''
  successMessage.value = ''
}

const clearFormError = () => {
  formError.value = ''
}

const clearPhoto = () => {
  photoFile.value =
    null

  photoPreview.value =
    ''

  if (cameraInput.value) {
    cameraInput.value.value =
      ''
  }

  if (galleryInput.value) {
    galleryInput.value.value =
      ''
  }
}

const handlePhotoSelected = (
  event,
) => {
  formError.value = ''

  const file =
    event.target.files?.[0]

  if (!file) {
    return
  }

  if (
    !file.type.startsWith(
      'image/',
    )
  ) {
    formError.value =
      'File harus berupa gambar'

    return
  }

  const maxSize =
    5 * 1024 * 1024

  if (
    file.size > maxSize
  ) {
    formError.value =
      'Ukuran foto maksimal 5 MB'

    return
  }

  photoFile.value =
    file

  const reader =
    new FileReader()

  reader.onload = () => {
    photoPreview.value =
      reader.result
  }

  reader.readAsDataURL(
    file,
  )
}

const openCamera = () => {
  cameraInput.value?.click()
}

const openGallery = () => {
  galleryInput.value?.click()
}

const resetForm = () => {
  form.warehouse_id = ''
  form.sku = ''
  form.name = ''

  form.classification =
    'REGULER'

  form.unit =
    'BOX'

  form.purchase_price =
    ''

  form.price =
    ''

  form.stock =
    ''

  form.min_stock =
    30

  form.barcode =
    ''

  selectedUnit.value =
    'BOX'

  measurementValue.value =
    ''

  measurementUnit.value =
    'ML'

  skuTouched.value =
    false

  formError.value =
    ''

  clearPhoto()
  closeDropdowns()
}

const loadProducts = async () => {
  loading.value =
    true

  errorMessage.value =
    ''

  try {
    products.value =
      await getProducts()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Gagal mengambil data produk'
  } finally {
    loading.value =
      false
  }
}

const openCreate = () => {
  clearMessage()
  resetForm()

  selectedProduct.value =
    null

  modalMode.value =
    'create'

  modalOpen.value =
    true
}

const openEdit = (
  product,
) => {
  clearMessage()
  resetForm()

  selectedProduct.value =
    product

  modalMode.value =
    'edit'

  form.sku =
    product.sku || ''

  const parsedMeasurement =
    splitProductNameMeasurement(
      product.name || '',
    )

  form.name =
    parsedMeasurement.name

  measurementValue.value =
    parsedMeasurement.value

  measurementUnit.value =
    parsedMeasurement.unit

  form.warehouse_id = product.warehouse_id || ''

  form.classification =
    product.classification ||
    'REGULER'

  form.unit =
    product.unit ||
    'BOX'

  form.purchase_price =
    Number(
      product.purchase_price ||
      0,
    )

  form.price =
    Number(
      product.price ||
      0,
    )

  form.stock =
    Number(
      product.stock ||
      0,
    )

  form.min_stock =
    Number(
      product.min_stock ??
      30,
    )

  form.barcode =
    product.barcode ||
    ''

  selectedUnit.value =
    product.unit ||
    'BOX'

  skuTouched.value =
    true

  if (
    product.image_url
  ) {
    photoPreview.value =
      product.image_url
  }

  modalOpen.value =
    true
}

const closeModal = () => {
  if (saving.value) {
    return
  }

  modalOpen.value =
    false

  selectedProduct.value =
    null

  formError.value =
    ''

  closeDropdowns()
}

const validateForm = () => {
  if (!form.warehouse_id) {
    return 'Pilih gudang penyimpanan produk'
  }

  if (!form.name.trim()) {
    return 'Nama produk wajib diisi'
  }

  if (!form.sku.trim()) {
    return 'SKU wajib diisi'
  }

  if (!form.unit.trim()) {
    return 'Unit wajib dipilih'
  }

  if (
    measurementValue.value !== '' &&
    Number(
      measurementValue.value,
    ) <= 0
  ) {
    return 'Ukuran / isi harus lebih dari 0'
  }

  if (
    canSeeInternalCost.value &&
    Number(
      form.purchase_price || 0,
    ) < 0
  ) {
    return 'Harga masuk tidak boleh negatif'
  }

  if (
    form.price === '' ||
    Number(form.price) < 0
  ) {
    return 'Harga jual wajib diisi dan tidak boleh negatif'
  }

  if (
    form.stock === '' ||
    Number(form.stock) < 0
  ) {
    return 'Stok saat ini wajib diisi dan tidak boleh negatif'
  }

  if (
    form.min_stock === '' ||
    Number(form.min_stock) < 0
  ) {
    return 'Batas pesan ulang wajib diisi dan tidak boleh negatif'
  }

  return null
}

const submitForm = async () => {
  formError.value =
    ''

  const validationError =
    validateForm()

  if (validationError) {
    formError.value =
      validationError

    return
  }

  saving.value =
    true

  const payload = {
    warehouse_id: form.warehouse_id,
    sku:
      form.sku
        .trim()
        .toUpperCase(),

    name:
      buildProductName(),

    classification:
      form.classification,

    unit:
      form.unit,

    price:
      Number(
        form.price || 0,
      ),

    stock:
      Number(
        form.stock || 0,
      ),

    min_stock:
      Number(
        form.min_stock ?? 30,
      ),

    barcode:
      form.barcode.trim(),
  }

  if (
    canSeeInternalCost.value
  ) {
    payload.purchase_price =
      Number(
        form.purchase_price || 0,
      )
  }

  try {
    if (
      modalMode.value ===
      'create'
    ) {
      await createProduct(
        payload,
      )

      successMessage.value =
        'Produk berhasil ditambahkan'
    } else {
      await updateProduct(
        selectedProduct.value.id,
        payload,
      )

      successMessage.value =
        'Produk berhasil diperbarui'
    }

    modalOpen.value =
      false

    selectedProduct.value =
      null

    await loadProducts()
  } catch (error) {
    const status =
      error.response?.status

    const backendMessage =
      error.response?.data?.error

    if (status === 401) {
      formError.value =
        'Sesi login sudah berakhir. Silakan login ulang.'
    } else if (status === 403) {
      formError.value =
        'Akun ini tidak memiliki izin untuk menyimpan produk.'
    } else if (status === 409) {
      formError.value =
        backendMessage ||
        'SKU atau barcode sudah digunakan produk lain.'
    } else {
      formError.value =
        backendMessage ||
        'Produk gagal disimpan. Periksa data lalu coba lagi.'
    }
  } finally {
    saving.value =
      false
  }
}

const toggleProductActive = async (
  product,
) => {
  if (
    !canManageProducts.value ||
    togglingProductID.value
  ) {
    return
  }

  clearMessage()

  togglingProductID.value =
    product.id

  try {
    await updateProduct(
      product.id,
      {
        is_active:
          !product.is_active,
      },
    )

    successMessage.value =
      product.is_active
        ? 'Produk berhasil dinonaktifkan'
        : 'Produk berhasil diaktifkan'

    await loadProducts()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Gagal mengubah status produk'
  } finally {
    togglingProductID.value =
      null
  }
}

const requestDelete = (
  product,
) => {
  if (
    !canDeleteProduct.value
  ) {
    return
  }

  pendingDeleteProduct.value =
    product

  confirmOpen.value =
    true
}

const cancelDelete = () => {
  if (
    deleting.value
  ) {
    return
  }

  confirmOpen.value =
    false

  pendingDeleteProduct.value =
    null
}

const confirmDelete = async () => {
  if (
    !pendingDeleteProduct.value ||
    deleting.value
  ) {
    return
  }

  clearMessage()

  deleting.value =
    true

  try {
    await deleteProduct(
      pendingDeleteProduct.value.id,
    )

    successMessage.value =
      'Produk berhasil dihapus permanen'

    confirmOpen.value =
      false

    pendingDeleteProduct.value =
      null

    await loadProducts()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Produk tidak dapat dihapus permanen'

    confirmOpen.value =
      false

    pendingDeleteProduct.value =
      null
  } finally {
    deleting.value =
      false
  }
}

const escapeHtml = (value) => {
  return String(
    value ?? '',
  )
    .replaceAll(
      '&',
      '&amp;',
    )
    .replaceAll(
      '<',
      '&lt;',
    )
    .replaceAll(
      '>',
      '&gt;',
    )
    .replaceAll(
      '"',
      '&quot;',
    )
    .replaceAll(
      "'",
      '&#039;',
    )
}

const openPrintWindow = (
  title,
  subtitle,
  tableHeader,
  tableBody,
) => {
  const printWindow =
    window.open(
      '',
      '_blank',
      'width=1200,height=850',
    )

  if (!printWindow) {
    errorMessage.value =
      'Browser memblokir jendela cetak'

    return
  }

  const dateText =
    new Intl.DateTimeFormat(
      'id-ID',
      {
        dateStyle: 'long',
        timeStyle: 'short',
      },
    ).format(
      new Date(),
    )

  printWindow.document.write(`
    <!DOCTYPE html>
    <html lang="id">
      <head>
        <meta charset="UTF-8">

        <title>${escapeHtml(title)}</title>

        <style>
          * {
            box-sizing: border-box;
          }

          body {
            margin: 0;
            padding: 32px;
            background: #fff;
            color: #101828;
            font-family: Arial, sans-serif;
          }

          .header {
            display: flex;
            justify-content: space-between;
            align-items: flex-start;
            gap: 24px;
            padding-bottom: 20px;
            border-bottom: 2px solid #101828;
          }

          .company {
            font-size: 20px;
            font-weight: 800;
          }

          .company-subtitle {
            margin-top: 5px;
            color: #667085;
            font-size: 11px;
          }

          .report-title {
            text-align: right;
            font-size: 18px;
            font-weight: 800;
          }

          .date {
            margin-top: 6px;
            text-align: right;
            color: #667085;
            font-size: 10px;
          }

          .subtitle {
            margin: 18px 0;
            padding: 11px 13px;
            border: 1px solid #e4e7ec;
            border-radius: 8px;
            background: #f9fafb;
            color: #667085;
            font-size: 10px;
          }

          table {
            width: 100%;
            border-collapse: collapse;
          }

          th {
            padding: 9px 7px;
            background: #f8fafc;
            border: 1px solid #eaecf0;
            text-align: left;
            font-size: 9px;
            text-transform: uppercase;
            color: #667085;
          }

          td {
            padding: 9px 7px;
            border: 1px solid #eaecf0;
            vertical-align: middle;
            font-size: 10px;
          }

          .name {
            font-weight: 700;
          }

          .code {
            margin-top: 3px;
            color: #667085;
            font-size: 9px;
          }

          .price,
          .number {
            text-align: right;
          }

          .photo {
            width: 42px;
            height: 42px;
            object-fit: cover;
            border-radius: 7px;
          }

          .placeholder {
            width: 42px;
            height: 42px;
            border-radius: 7px;
            display: flex;
            align-items: center;
            justify-content: center;
            background: #ecfdf3;
            color: #067647;
            font-size: 9px;
            font-weight: 800;
          }

          .footer {
            margin-top: 20px;
            padding-top: 12px;
            border-top: 1px solid #eaecf0;
            display: flex;
            justify-content: space-between;
            color: #98a2b3;
            font-size: 9px;
          }

          @page {
            size: A4 landscape;
            margin: 12mm;
          }

          @media print {
            body {
              padding: 0;
            }

            tr {
              break-inside: avoid;
            }
          }
        </style>
      </head>

      <body>
        <div class="header">
          <div>
            <div class="company">
              PT. FURI ANUGERAH SEJAHTERA
            </div>

            <div class="company-subtitle">
              Business Operations Platform
            </div>
          </div>

          <div>
            <div class="report-title">
              ${escapeHtml(title)}
            </div>

            <div class="date">
              ${escapeHtml(dateText)}
            </div>
          </div>
        </div>

        <div class="subtitle">
          ${escapeHtml(subtitle)}
        </div>

        <table>
          <thead>
            ${tableHeader}
          </thead>

          <tbody>
            ${tableBody}
          </tbody>
        </table>

        <div class="footer">
          <span>
            PT. Furi Anugerah Sejahtera
          </span>

          <span>
            ${filteredProducts.value.length} produk
          </span>
        </div>

        <script>
          window.addEventListener(
            'load',
            () => {
              setTimeout(() => {
                window.print()
              }, 250)
            }
          )
        <\/script>
      </body>
    </html>
  `)

  printWindow.document.close()

  exportMenuOpen.value =
    false
}

const printCatalog = async () => {
  if (!catalogProducts.value.length) { errorMessage.value = 'Tidak ada produk aktif untuk dicetak'; return }
  const products = catalogProducts.value.map(product => ({ ...product }))
  const popup = window.open('', '_blank')
  if (!popup) { errorMessage.value = 'Izinkan pop-up browser untuk mencetak katalog'; return }
  popup.opener = null
  popup.document.write('<p>Menyiapkan katalog produk…</p>')
  try {
    await loadCompanyProfile()
    const html = buildProductCatalogHTML(products, { ...companyProfile }, formatCurrency, productWarehouseLabel)
    writeProductCatalogPrint(popup, html)
    exportMenuOpen.value = false
  } catch (error) { popup.close(); errorMessage.value = error.response?.data?.error || error.message || 'Katalog belum dapat dicetak.' }
}

const printStockReport = () => {
  const body =
    filteredProducts.value
      .map(
        (product, index) => `
          <tr>
            <td>${index + 1}</td>

            <td>
              <div class="name">
                ${escapeHtml(product.name)}
              </div>

              <div class="code">
                ${escapeHtml(product.product_code)}
              </div>
            </td>

            <td>
              ${escapeHtml(product.sku)}
            </td>

            <td>
              ${escapeHtml(product.barcode || '-')}
            </td>

            <td>
              ${escapeHtml(product.unit)}
            </td>

            <td class="number">
              ${escapeHtml(product.stock)}
            </td>

            <td class="number">
              ${escapeHtml(product.min_stock)}
            </td>

            <td>
              ${product.is_active ? 'ACTIVE' : 'INACTIVE'}
            </td>
          </tr>
        `,
      )
      .join('')

  openPrintWindow(
    'STOCK REPORT',
    'Laporan stok operasional gudang.',
    `
      <tr>
        <th>No</th>
        <th>Produk</th>
        <th>SKU</th>
        <th>Barcode</th>
        <th>Unit</th>
        <th>Stock</th>
        <th>Minimum</th>
        <th>Status</th>
      </tr>
    `,
    body,
  )
}

const printInternalReport = () => {
  if (!isAdmin.value) {
    return
  }

  const body =
    filteredProducts.value
      .map(
        (product, index) => {
          const margin =
            marginAmountFor(product)

          const marginRate =
            marginPercentageFor(product)

          return `
            <tr>
              <td>${index + 1}</td>

              <td>
                <div class="name">
                  ${escapeHtml(product.name)}
                </div>

                <div class="code">
                  ${escapeHtml(product.product_code)}
                </div>
              </td>

              <td>
                ${escapeHtml(product.sku)}
              </td>

              <td>
                ${escapeHtml(product.barcode || '-')}
              </td>

              <td>
                ${escapeHtml(productWarehouseLabel(product))}
              </td>

              <td>
                ${escapeHtml(product.unit)}
              </td>

              <td class="price">
                ${
                  Number(product.purchase_price || 0) > 0
                    ? escapeHtml(formatCurrency(product.purchase_price))
                    : '-'
                }
              </td>

              <td class="price">
                ${escapeHtml(formatCurrency(product.price))}
              </td>

              <td class="price">
                ${
                  margin === null
                    ? '-'
                    : escapeHtml(formatCurrency(margin))
                }
              </td>

              <td class="number">
                ${
                  marginRate === null
                    ? '-'
                    : `${marginRate.toFixed(1)}%`
                }
              </td>

              <td class="number">
                ${escapeHtml(product.stock)}
              </td>
            </tr>
          `
        },
      )
      .join('')

  openPrintWindow(
    'INTERNAL INVENTORY REPORT',
    'Laporan internal Owner/Admin. Berisi harga pembelian, harga jual dan margin perusahaan.',
    `
      <tr>
        <th>No</th>
        <th>Produk</th>
        <th>SKU</th>
        <th>Barcode</th>
        <th>Gudang</th>
        <th>Unit</th>
        <th>Harga Masuk</th>
        <th>Harga Jual</th>
        <th>Margin</th>
        <th>Rate</th>
        <th>Stock</th>
      </tr>
    `,
    body,
  )
}


const warehouses = ref([])
const warehouseLoading = ref(false)
const warehouseCreating = ref(false)
const newWarehouseNumber = ref(4)

const warehouseLabel = warehouse => {
  if (warehouse.warehouse_code === 'GUDANG_1') {
    return 'Prekusor'
  }
  return warehouse.name
}

const activeWarehouses = computed(() => {
  return warehouses.value.filter(warehouse => warehouse.is_active).sort((a, b) => {
    const number = warehouse => warehouse.warehouse_code === 'GUDANG_1'
      ? 1
      : Number((warehouse.name.match(/\d+/) || ['999'])[0])
    return number(a) - number(b)
  })
})

const classificationFilterOptions = computed(() => [
  { value: 'ALL', label: 'Semua gudang' },
  ...activeWarehouses.value.map(warehouse => ({ value: warehouse.id, label: warehouseLabel(warehouse) })),
  { value: 'UNASSIGNED', label: 'Gudang belum diatur' },
])

const productWarehouseLabel = product => {
  const warehouse = warehouses.value.find(item => item.id === product.warehouse_id)
  return warehouse ? warehouseLabel(warehouse) : 'Gudang belum diatur'
}

const loadWarehouses = async () => {
  warehouseLoading.value = true
  try {
    warehouses.value = await getWarehouses()
    newWarehouseNumber.value = Math.max(4, ...warehouses.value.map(item => Number((item.name.match(/\d+/) || ['0'])[0]) + 1))
  } catch (error) {
    errorMessage.value = error.response?.data?.error || 'Master gudang belum dapat dimuat'
  } finally {
    warehouseLoading.value = false
  }
}

const selectProductWarehouse = () => {
  const warehouse = activeWarehouses.value.find(item => item.id === form.warehouse_id)
  if (warehouse) {
    form.classification = warehouse.classification
  }
}

const addManualWarehouse = async () => {
  const number = Number(newWarehouseNumber.value)
  if (!Number.isInteger(number) || number < 4 || number > 999) {
    formError.value = 'Nomor gudang manual harus 4 sampai 999'
    return
  }
  const name = `Gudang ${number}`
  if (warehouses.value.some(item => item.name.toLowerCase() === name.toLowerCase())) {
    formError.value = `${name} sudah terdaftar. Pilih dari daftar atau aktifkan melalui admin.`
    return
  }
  warehouseCreating.value = true
  formError.value = ''
  try {
    const created = await createWarehouse({ name, classification: 'REGULER', address: '' })
    await loadWarehouses()
    form.warehouse_id = created.id
    selectProductWarehouse()
  } catch (error) {
    formError.value = error.response?.data?.error || 'Gudang baru belum dapat disimpan'
  } finally {
    warehouseCreating.value = false
  }
}

onMounted(() => {
  loadProducts()
  loadWarehouses()

  window.addEventListener(
    'click',
    closeDropdowns,
  )
})

onBeforeUnmount(() => {
  window.removeEventListener(
    'click',
    closeDropdowns,
  )
})
</script>

<template>
  <div class="inventory-page p-5 sm:p-6 lg:p-8 xl:p-10">
    <div class="inventory-heading">
      <div>
        <p class="inventory-kicker">
          Inventory Management
        </p>

        <h2 class="inventory-title">
          Product Master
        </h2>

        <p class="inventory-description">
          Kelola produk, harga, stok,
          klasifikasi dan identitas barang
          PT. FAS.
        </p>
      </div>

      <div class="heading-actions">
        <div
          class="export-wrapper"
          @click.stop
        >
          <button
            type="button"
            class="catalog-button"
            @click="toggleExportMenu"
          >
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              class="h-4 w-4"
            >
              <path d="M6 9V2h12v7" />

              <path
                d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"
              />

              <rect
                x="6"
                y="14"
                width="12"
                height="8"
                rx="1"
              />
            </svg>

            Cetak / PDF

            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              class="h-4 w-4"
            >
              <path d="m6 9 6 6 6-6" />
            </svg>
          </button>

          <Transition name="dropdown">
            <div
              v-if="exportMenuOpen"
              class="export-menu"
            >
              <button
                type="button"
                class="export-option"
                @click="printCatalog"
              >
                <div>
                  <p class="export-title">
                    Product Catalog
                  </p>

                  <p class="export-subtitle">
                    Sales & customer
                  </p>
                </div>
              </button>

              <button
                type="button"
                class="export-option"
                @click="printStockReport"
              >
                <div>
                  <p class="export-title">
                    Stock Report
                  </p>

                  <p class="export-subtitle">
                    Operasional gudang
                  </p>
                </div>
              </button>

              <button
                v-if="isAdmin"
                type="button"
                class="export-option internal-option"
                @click="printInternalReport"
              >
                <div>
                  <p class="export-title">
                    Internal Inventory
                  </p>

                  <p class="export-subtitle">
                    Harga masuk & margin
                  </p>
                </div>

                <span class="owner-badge">
                  OWNER
                </span>
              </button>
            </div>
          </Transition>
        </div>

        <button
          v-if="canManageProducts"
          type="button"
          class="add-product-button"
          @click="openCreate"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            class="h-4 w-4"
          >
            <path d="M12 5v14" />
            <path d="M5 12h14" />
          </svg>

          Tambah Produk
        </button>
      </div>
    </div>

    <div
      v-if="errorMessage"
      class="page-error"
    >
      {{ errorMessage }}
    </div>

    <div
      v-if="successMessage"
      class="page-success"
    >
      {{ successMessage }}
    </div>

    <section class="summary-grid">
      <article class="summary-card">
        <p class="summary-label">
          Total Products
        </p>

        <p class="summary-value">
          {{ totalProducts }}
        </p>

        <p class="summary-footer">
          Product master
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Active Products
        </p>

        <p class="summary-value">
          {{ activeProducts }}
        </p>

        <p class="summary-footer">
          Produk aktif
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Prekursor
        </p>

        <p class="summary-value">
          {{ precursorProducts }}
        </p>

        <p class="summary-footer">
          Produk terkendali
        </p>
      </article>

      <article class="summary-card">
        <p class="summary-label">
          Perlu Dipesan
        </p>

        <p class="summary-value">
          {{ lowStockProducts }}
        </p>

        <p class="summary-footer">
          Stok menyentuh batas pesan
        </p>
      </article>
    </section>

    <section class="inventory-panel">
      <div class="inventory-toolbar">
        <div class="search-box">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            class="search-icon"
          >
            <circle
              cx="11"
              cy="11"
              r="7"
            />

            <path d="m20 20-3-3" />
          </svg>

          <input
            v-model="search"
            type="text"
            placeholder="Cari kode, SKU, nama atau barcode..."
            class="search-input"
          />
        </div>

        <div
          class="custom-select"
          @click.stop
        >
          <button
            type="button"
            class="custom-select-trigger"
            @click="
              toggleClassificationFilterMenu
            "
          >
            <span>
              {{
                selectedClassificationFilterLabel
              }}
            </span>

            <span class="chevron">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </span>
          </button>

          <Transition name="dropdown">
            <div
              v-if="
                classificationFilterMenuOpen
              "
              class="custom-select-menu"
            >
              <button
                v-for="
                  item
                  in classificationFilterOptions
                "
                :key="item.value"
                type="button"
                :class="['custom-option', { 'custom-option-active': classificationFilter === item.value }]"
                @click="
                  selectClassificationFilter(
                    item.value,
                  )
                "
              >
                {{ item.label }}
              </button>
            </div>
          </Transition>
        </div>

        <div
          class="custom-select"
          @click.stop
        >
          <button
            type="button"
            class="custom-select-trigger"
            @click="
              toggleStatusFilterMenu
            "
          >
            <span>
              {{
                selectedStatusFilterLabel
              }}
            </span>

            <span class="chevron">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </span>
          </button>

          <Transition name="dropdown">
            <div
              v-if="
                statusFilterMenuOpen
              "
              class="custom-select-menu"
            >
              <button
                v-for="
                  item
                  in statusFilterOptions
                "
                :key="item.value"
                type="button"
                :class="['custom-option', { 'custom-option-active': statusFilter === item.value }]"
                @click="
                  selectStatusFilter(
                    item.value,
                  )
                "
              >
                {{ item.label }}
              </button>
            </div>
          </Transition>
        </div>

        <button
          type="button"
          class="refresh-button"
          title="Refresh"
          @click="loadProducts"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            class="h-4 w-4"
          >
            <path
              d="M20 11a8.1 8.1 0 0 0-15.5-2M4 4v5h5"
            />

            <path
              d="M4 13a8.1 8.1 0 0 0 15.5 2M20 20v-5h-5"
            />
          </svg>
        </button>
      </div>

      <div
        v-if="loading"
        class="loading-area"
      >
        <div class="loading-ring"></div>
      </div>

      <div
        v-else-if="
          filteredProducts.length === 0
        "
        class="empty-area"
      >
        Produk tidak ditemukan
      </div>

      <div
        v-else
        class="inventory-table-scroll hidden md:block"
      >
        <table class="inventory-table">
          <thead>
            <tr>
              <th>Product</th>
              <th>Code</th>
              <th>SKU</th>
              <th>Barcode</th>
              <th>Gudang</th>
              <th>Unit</th>

              <th
                v-if="canSeeInternalCost"
                class="price-column"
              >
                Harga Masuk
              </th>

              <th class="price-column">
                Harga Jual
              </th>

              <th
                v-if="canSeeInternalCost"
                class="price-column"
              >
                Margin
              </th>

              <th>Stok Saat Ini</th>
              <th>Kondisi Stok</th>
              <th>Status</th>
              <th>Action</th>
            </tr>
          </thead>

          <tbody>
            <tr
              v-for="
                product
                in filteredProducts
              "
              :key="product.id"
            >
              <td>
                <div class="product-cell">
                  <div class="product-thumbnail">
                    <img
                      v-if="
                        product.image_url
                      "
                      :src="
                        product.image_url
                      "
                      :alt="
                        product.name
                      "
                      class="h-full w-full object-cover"
                    />

                    <span v-else>
                      FAS
                    </span>
                  </div>

                  <span class="product-name">
                    {{ product.name }}
                  </span>
                </div>
              </td>

              <td>
                <span class="code-badge">
                  {{
                    product.product_code
                  }}
                </span>
              </td>

              <td>
                <span class="sku-badge">
                  {{ product.sku }}
                </span>
              </td>

              <td>
                <span class="barcode-value">
                  {{
                    product.barcode ||
                    '—'
                  }}
                </span>
              </td>

              <td>
                <span
                  :class="[
                    'classification-badge',
                    product.classification ===
                    'PREKURSOR'
                      ? 'classification-precursor'
                      : 'classification-regular',
                  ]"
                >
                  {{ productWarehouseLabel(product) }}
                </span>
              </td>

              <td>
                {{ product.unit }}
              </td>

              <td
                v-if="canSeeInternalCost"
                class="price-column"
              >
                {{
                  Number(
                    product.purchase_price ||
                    0,
                  ) > 0
                    ? formatCurrency(
                        product.purchase_price,
                      )
                    : '—'
                }}
              </td>

              <td class="price-column selling-price">
                {{
                  formatCurrency(
                    product.price,
                  )
                }}
              </td>

              <td
                v-if="canSeeInternalCost"
                class="price-column"
              >
                <template
                  v-if="
                    marginAmountFor(
                      product,
                    ) !== null
                  "
                >
                  <p class="margin-table-value">
                    {{
                      formatCurrency(
                        marginAmountFor(
                          product,
                        ),
                      )
                    }}
                  </p>

                  <p class="margin-table-rate">
                    {{
                      marginPercentageFor(
                        product,
                      ).toFixed(1)
                    }}%
                  </p>
                </template>

                <span v-else>
                  —
                </span>
              </td>

              <td>
                <p
                  :class="[
                    'stock-value',
                    stockState(product).label !==
                    'AMAN'
                      ? 'stock-low'
                      : '',
                  ]"
                >
                  {{ product.stock }}
                  {{ product.unit }}
                </p>

                <p class="stock-min">
                  Batas
                  {{ product.min_stock ?? 30 }}
                  {{ product.unit }}
                </p>
              </td>

              <td>
                <span
                  :class="[
                    'stock-status',
                    stockState(product).className,
                  ]"
                >
                  {{ stockState(product).label }}
                </span>
              </td>

              <td>
                <span
                  :class="[
                    'status-badge',
                    product.is_active
                      ? 'status-active'
                      : 'status-inactive',
                  ]"
                >
                  {{
                    product.is_active
                      ? 'Active'
                      : 'Inactive'
                  }}
                </span>
              </td>

              <td>
                <div class="action-group">
                  <button
                    v-if="canManageProducts"
                    type="button"
                    class="table-button"
                    title="Edit"
                    @click="
                      openEdit(
                        product,
                      )
                    "
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="1.8"
                      class="h-4 w-4"
                    >
                      <path d="M12 20h9" />

                      <path
                        d="M16.5 3.5a2.1 2.1 0 0 1 3 3L8 18l-4 1 1-4Z"
                      />
                    </svg>
                  </button>

                  <button
                    v-if="canManageProducts"
                    type="button"
                    :class="[
                      'table-button-toggle',
                      product.is_active
                        ? 'toggle-off'
                        : 'toggle-on',
                    ]"
                    :title="
                      product.is_active
                        ? 'Nonaktifkan'
                        : 'Aktifkan'
                    "
                    :disabled="
                      togglingProductID ===
                      product.id
                    "
                    @click="
                      toggleProductActive(
                        product,
                      )
                    "
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="1.8"
                      class="h-4 w-4"
                    >
                      <path d="M12 2v10" />
                      <path
                        d="M6.4 5.6a8 8 0 1 0 11.2 0"
                      />
                    </svg>
                  </button>

                  <button
                    v-if="canDeleteProduct"
                    type="button"
                    class="table-button-danger"
                    title="Hapus Permanen"
                    @click="
                      requestDelete(
                        product,
                      )
                    "
                  >
                    <svg
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      stroke-width="1.8"
                      class="h-4 w-4"
                    >
                      <path d="M3 6h18" />
                      <path d="M8 6V4h8v2" />
                      <path d="M19 6l-1 14H6L5 6" />
                      <path d="M10 11v5" />
                      <path d="M14 11v5" />
                    </svg>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div
        v-if="
          !loading &&
          filteredProducts.length > 0
        "
        class="mobile-list md:hidden"
      >
        <article
          v-for="
            product
            in filteredProducts
          "
          :key="product.id"
          class="mobile-product"
        >
          <div class="flex gap-3">
            <div class="mobile-thumbnail">
              <img
                v-if="
                  product.image_url
                "
                :src="
                  product.image_url
                "
                :alt="
                  product.name
                "
                class="h-full w-full object-cover"
              />

              <span v-else>
                FAS
              </span>
            </div>

            <div class="min-w-0 flex-1">
              <p class="product-name">
                {{ product.name }}
              </p>

              <p class="mobile-code">
                {{
                  product.product_code
                }}
              </p>

              <p class="mobile-sku">
                SKU:
                {{ product.sku }}
              </p>
            </div>
          </div>

          <div class="mobile-grid">
            <div>
              <p class="mobile-label">
                Unit
              </p>

              <p class="mobile-value">
                {{ product.unit }}
              </p>
            </div>

            <div>
              <p class="mobile-label">
                Stok Saat Ini
              </p>

              <p class="mobile-value">
                {{ product.stock }}
                {{ product.unit }}
              </p>

              <p class="mobile-stock-limit">
                Batas
                {{ product.min_stock ?? 30 }}
                {{ product.unit }}
              </p>
            </div>

            <div>
              <p class="mobile-label">
                Kondisi Stok
              </p>

              <span
                :class="[
                  'stock-status',
                  stockState(product).className,
                ]"
              >
                {{ stockState(product).label }}
              </span>
            </div>

            <div>
              <p class="mobile-label">
                Harga Jual
              </p>

              <p class="mobile-value">
                {{
                  formatCurrency(
                    product.price,
                  )
                }}
              </p>
            </div>

            <div>
              <p class="mobile-label">
                Barcode
              </p>

              <p class="mobile-value">
                {{
                  product.barcode ||
                  '—'
                }}
              </p>
            </div>

            <div
              v-if="canSeeInternalCost"
            >
              <p class="mobile-label">
                Harga Masuk
              </p>

              <p class="mobile-value">
                {{
                  Number(
                    product.purchase_price ||
                    0,
                  ) > 0
                    ? formatCurrency(
                        product.purchase_price,
                      )
                    : '—'
                }}
              </p>
            </div>

            <div
              v-if="canSeeInternalCost"
            >
              <p class="mobile-label">
                Margin
              </p>

              <p class="mobile-value">
                {{
                  marginAmountFor(
                    product,
                  ) !== null
                    ? formatCurrency(
                        marginAmountFor(
                          product,
                        ),
                      )
                    : '—'
                }}
              </p>
            </div>
          </div>

          <div class="mobile-actions">
            <button
              v-if="canManageProducts"
              type="button"
              class="mobile-action"
              @click="
                openEdit(
                  product,
                )
              "
            >
              Edit
            </button>

            <button
              v-if="canManageProducts"
              type="button"
              class="mobile-action-toggle"
              :disabled="
                togglingProductID ===
                product.id
              "
              @click="
                toggleProductActive(
                  product,
                )
              "
            >
              {{
                product.is_active
                  ? 'Nonaktifkan'
                  : 'Aktifkan'
              }}
            </button>

            <button
              v-if="canDeleteProduct"
              type="button"
              class="mobile-action-danger"
              @click="
                requestDelete(
                  product,
                )
              "
            >
              Hapus
            </button>
          </div>
        </article>
      </div>
    </section>

    <Transition name="modal">
      <div
        v-if="modalOpen"
        class="modal-shell"
      >
        <div
          class="modal-backdrop"
          @click="closeModal"
        ></div>

        <section class="product-modal">
          <div class="modal-header">
            <div>
              <p class="modal-eyebrow">
                Product Master
              </p>

              <h3 class="modal-title">
                {{
                  modalMode === 'create'
                    ? 'Tambah Produk'
                    : 'Edit Produk'
                }}
              </h3>
            </div>

            <button
              type="button"
              class="modal-close"
              @click="closeModal"
            >
              <svg
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="1.8"
                class="h-5 w-5"
              >
                <path d="m6 6 12 12" />
                <path d="m18 6-12 12" />
              </svg>
            </button>
          </div>

          <form
            class="product-form-scroll"
            @submit.prevent="submitForm"
          >
            <div
              v-if="formError"
              class="form-error"
            >
              <div class="form-error-icon">
                !
              </div>

              <div>
                <p class="form-error-title">
                  Produk belum tersimpan
                </p>

                <p class="form-error-text">
                  {{ formError }}
                </p>
              </div>
            </div>

            <section class="photo-section">
              <div class="photo-preview">
                <img
                  v-if="photoPreview"
                  :src="photoPreview"
                  alt="Preview"
                  class="h-full w-full object-cover"
                />

                <div
                  v-else
                  class="photo-empty"
                >
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.5"
                    class="h-7 w-7"
                  >
                    <path
                      d="M4 7h3l2-3h6l2 3h3v13H4V7Z"
                    />

                    <circle
                      cx="12"
                      cy="13"
                      r="4"
                    />
                  </svg>

                  <span>
                    FOTO
                  </span>
                </div>
              </div>

              <div class="flex-1">
                <p class="form-label">
                  Foto Produk
                </p>

                <p class="photo-description">
                  Ambil foto dari kamera HP
                  atau pilih dari galeri.
                </p>

                <div class="photo-actions">
                  <button
                    type="button"
                    class="photo-button"
                    @click="
                      openCamera
                    "
                  >
                    Kamera
                  </button>

                  <button
                    type="button"
                    class="photo-button"
                    @click="
                      openGallery
                    "
                  >
                    Galeri
                  </button>

                  <button
                    v-if="photoPreview"
                    type="button"
                    class="photo-remove"
                    @click="
                      clearPhoto
                    "
                  >
                    Hapus
                  </button>
                </div>

                <input
                  ref="cameraInput"
                  type="file"
                  accept="image/*"
                  capture="environment"
                  class="hidden"
                  @change="
                    handlePhotoSelected
                  "
                />

                <input
                  ref="galleryInput"
                  type="file"
                  accept="image/*"
                  class="hidden"
                  @change="
                    handlePhotoSelected
                  "
                />
              </div>
            </section>

            <div class="form-grid">
              <div>
                <label class="form-label">
                  Product Code
                </label>

                <div class="generated-field">
                  <div>
                    <p class="generated-value">
                      {{
                        modalMode === 'edit'
                          ? selectedProduct?.product_code
                          : 'AUTO'
                      }}
                    </p>

                    <p class="generated-sub">
                      Generated by system
                    </p>
                  </div>
                </div>
              </div>

              <div>
                <div class="form-label-row">
                  <label class="form-label">
                    SKU
                  </label>

                  <button
                    type="button"
                    class="sku-auto-button"
                    @click="
                      regenerateSKU
                    "
                  >
                    AUTO
                  </button>
                </div>

                <input
                  v-model="form.sku"
                  required
                  type="text"
                  autocomplete="off"
                  class="form-input"
                  placeholder="Auto SKU"
                  @input="
                    handleSKUInput
                  "
                />
              </div>

              <div class="span-two">
                <label class="form-label">
                  Nama Produk
                </label>

                <input
                  v-model="form.name"
                  required
                  type="text"
                  autocomplete="off"
                  class="form-input"
                  placeholder="Contoh: Sanmol Syrup"
                  @input="
                    handleNameInput
                  "
                />
              </div>

              <div>
                <label class="form-label">Gudang Penyimpanan</label>
                <FasSelect v-model="form.warehouse_id" :disabled="warehouseLoading || warehouseCreating" aria-label="Gudang penyimpanan produk" @change="selectProductWarehouse">
                  <option value="" disabled>Pilih gudang</option>
                  <option v-for="warehouse in activeWarehouses" :key="warehouse.id" :value="warehouse.id">
                    {{ warehouseLabel(warehouse) }}
                  </option>
                </FasSelect>
                <p class="field-helper">Gudang 1 bernama Prekusor. Pilih Gudang 2, Gudang 3, atau tambahkan gudang berikutnya.</p>
                <div v-if="canManageProducts" class="manual-warehouse">
                  <label>
                    <span>Nomor gudang baru</span>
                    <input v-model.number="newWarehouseNumber" type="number" min="4" max="999" class="form-input" aria-label="Nomor gudang baru" />
                  </label>
                  <button type="button" class="warehouse-add-button" :disabled="warehouseCreating || warehouseLoading" @click="addManualWarehouse">
                    {{ warehouseCreating ? 'Menyimpan...' : 'Tambah Gudang ' + newWarehouseNumber }}
                  </button>
                </div>
              </div>

              <div>
                <label class="form-label">
                  Unit Barang
                </label>

                <div
                  class="custom-select"
                  @click.stop
                >
                  <button
                    type="button"
                    class="custom-select-trigger"
                    @click="
                      toggleUnitMenu
                    "
                  >
                    <span>
                      {{ selectedUnit }}
                    </span>

                    <span class="chevron">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </span>
                  </button>

                  <Transition name="dropdown">
                    <div
                      v-if="unitMenuOpen"
                      class="custom-select-menu"
                    >
                      <button
                        v-for="
                          unit
                          in unitOptions
                        "
                        :key="unit"
                        type="button"
                        :class="['custom-option', { 'custom-option-active': selectedUnit === unit }]"
                        @click="
                          selectUnit(unit)
                        "
                      >
                        {{ unit }}
                      </button>
                    </div>
                  </Transition>
                </div>
              </div>

              <div class="span-two">
                <label class="form-label">
                  Ukuran / Isi
                  <span class="optional-label">
                    Optional
                  </span>
                </label>

                <div class="measurement-row">
                  <input
                    v-model="
                      measurementValue
                    "
                    type="number"
                    min="0"
                    step="any"
                    inputmode="decimal"
                    class="form-input measurement-input"
                    placeholder="Contoh: 27, 35, 62.5, 500"
                    @input="
                      handleMeasurementInput
                    "
                  />

                  <div
                    class="custom-select measurement-unit"
                    @click.stop
                  >
                    <button
                      type="button"
                      class="custom-select-trigger"
                      @click="
                        toggleMeasurementUnitMenu
                      "
                    >
                      <span>
                        {{
                          measurementUnit
                        }}
                      </span>

                      <span class="chevron">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
                <path d="m6 9 6 6 6-6" />
              </svg>
            </span>
                    </button>

                    <Transition name="dropdown">
                      <div
                        v-if="
                          measurementUnitMenuOpen
                        "
                        class="custom-select-menu"
                      >
                        <button
                          v-for="
                            unit
                            in measurementUnitOptions
                          "
                          :key="unit"
                          type="button"
                          :class="['custom-option', { 'custom-option-active': measurementUnit === unit }]"
                          @click="
                            selectMeasurementUnit(
                              unit,
                            )
                          "
                        >
                          {{ unit }}
                        </button>
                      </div>
                    </Transition>
                  </div>
                </div>

                <div class="quick-size-list">
                  <button
                    v-for="
                      value
                      in quickMeasurements
                    "
                    :key="value"
                    type="button"
                    :class="[
                      'quick-size',
                      Number(
                        measurementValue,
                      ) === value
                        ? 'quick-size-active'
                        : '',
                    ]"
                    @click="
                      useQuickMeasurement(
                        value,
                      )
                    "
                  >
                    {{ value }}
                  </button>
                </div>

                <p class="field-helper">
                  Angka bebas. Contoh:
                  27 ML, 35 ML, 500 MG,
                  1.5 L.
                </p>
              </div>

              <div>
                <label class="form-label">
                  Stok Saat Ini
                  ({{ stockUnitLabel }})
                </label>

                <input
                  v-model.number="
                    form.stock
                  "
                  required
                  type="number"
                  min="0"
                  step="1"
                  class="form-input stock-input"
                  placeholder="Contoh: 200"
                />

                <p class="field-helper">
                  Jumlah barang yang tersedia sekarang.
                </p>
              </div>

              <div>
                <label class="form-label">
                  Batas Pesan Ulang
                  ({{ stockUnitLabel }})
                </label>

                <input
                  v-model.number="
                    form.min_stock
                  "
                  required
                  type="number"
                  min="0"
                  step="1"
                  class="form-input"
                  placeholder="30"
                />

                <p class="field-helper">
                  Default 30. Boleh diubah sesuai kebutuhan produk.
                </p>
              </div>

              <div
                v-if="
                  canSeeInternalCost
                "
              >
                <label class="form-label">
                  Harga Masuk (Rp / {{ stockUnitLabel }})
                </label>

                <input
                  v-model.number="
                    form.purchase_price
                  "
                  type="number"
                  min="0"
                  step="1"
                  class="form-input"
                  placeholder="Contoh: 42500"
                />
              </div>

              <div>
                <label class="form-label">
                  Harga Jual (Rp / {{ stockUnitLabel }})
                </label>

                <input
                  v-model.number="
                    form.price
                  "
                  required
                  type="number"
                  min="0"
                  step="1"
                  class="form-input"
                  placeholder="Contoh: 46000"
                />
              </div>

              <div
                v-if="
                  canSeeInternalCost
                "
                class="span-two"
              >
                <div class="margin-card">
                  <div>
                    <p class="margin-label">
                      Gross Margin
                    </p>

                    <p class="margin-value">
                      {{
                        formatCurrency(
                          marginAmount,
                        )
                      }}
                    </p>
                  </div>

                  <div class="text-right">
                    <p class="margin-label">
                      Margin Rate
                    </p>

                    <p
                      :class="[
                        'margin-percent',
                        marginAmount < 0
                          ? 'margin-negative'
                          : '',
                      ]"
                    >
                      {{
                        marginPercentage.toFixed(
                          1,
                        )
                      }}%
                    </p>
                  </div>
                </div>
              </div>

              <div class="span-two">
                <div class="form-label-row">
                  <label class="form-label">
                    Barcode
                  </label>

                  <span class="optional-label">
                    Optional
                  </span>
                </div>

                <input
                  v-model="form.barcode"
                  type="text"
                  autocomplete="off"
                  class="form-input"
                  placeholder="Kosongkan jika produk tidak memiliki barcode"
                />

                <p class="field-helper">
                  Barcode tidak wajib.
                  USB scanner dapat langsung
                  membaca ke field ini.
                </p>
              </div>
            </div>

            <div class="form-footer">
              <button
                type="button"
                class="secondary-button"
                :disabled="saving"
                @click="
                  closeModal
                "
              >
                Batal
              </button>

              <button
                type="submit"
                class="primary-button"
                :disabled="saving"
              >
                <span
                  v-if="saving"
                  class="button-spinner"
                ></span>

                {{
                  saving
                    ? 'Menyimpan...'
                    : modalMode ===
                        'create'
                      ? 'Simpan Produk'
                      : 'Simpan Perubahan'
                }}
              </button>
            </div>
          </form>
        </section>
      </div>
    </Transition>

    <Transition name="confirm">
      <div
        v-if="confirmOpen"
        class="confirm-shell"
      >
        <div
          class="modal-backdrop"
          @click="
            cancelDelete
          "
        ></div>

        <section class="confirm-dialog">
          <div class="confirm-icon">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="1.8"
              class="h-6 w-6"
            >
              <path d="M3 6h18" />
              <path d="M8 6V4h8v2" />
              <path d="M19 6l-1 14H6L5 6" />
              <path d="M10 11v5" />
              <path d="M14 11v5" />
            </svg>
          </div>

          <p class="confirm-eyebrow">
            Product Management
          </p>

          <h3 class="confirm-title">
            Hapus Produk Permanen?
          </h3>

          <p class="confirm-description">
            Produk
            <strong>
              {{
                pendingDeleteProduct?.name
              }}
            </strong>
            akan dicoba dihapus permanen dari Product Master.
          </p>

          <div class="confirm-product">
            <div>
              <p class="confirm-product-name">
                {{
                  pendingDeleteProduct?.name
                }}
              </p>

              <p class="confirm-product-code">
                {{
                  pendingDeleteProduct?.product_code
                }}
                ·
                {{
                  pendingDeleteProduct?.sku
                }}
              </p>
            </div>

            <span class="confirm-status">
              DELETE
            </span>
          </div>

          <p class="confirm-note">
            Jika produk sudah pernah dipakai pada invoice,
            stock movement, atau histori transaksi lain,
            server akan menolak penghapusan permanen.
            Gunakan Nonaktifkan untuk produk yang masih punya histori.
          </p>

          <div class="confirm-actions">
            <button
              type="button"
              class="confirm-cancel"
              :disabled="
                deleting
              "
              @click="
                cancelDelete
              "
            >
              Batal
            </button>

            <button
              type="button"
              class="confirm-danger"
              :disabled="
                deleting
              "
              @click="
                confirmDelete
              "
            >
              <span
                v-if="
                  deleting
                "
                class="button-spinner"
              ></span>

              {{
                deleting
                  ? 'Menghapus...'
                  : 'Ya, Hapus Permanen'
              }}
            </button>
          </div>
        </section>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.inventory-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1.5rem;
}

.inventory-kicker {
  font-size: 0.8rem;
  font-weight: 700;
  color: var(--fas-accent-text);
}

.inventory-title {
  margin-top: 0.45rem;
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.025em;
  color: var(--fas-text);
}

.inventory-description {
  margin-top: 0.75rem;
  max-width: 42rem;
  font-size: 0.82rem;
  line-height: 1.5rem;
  color: var(--fas-muted);
}

.heading-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.export-wrapper {
  position: relative;
}

.catalog-button,
.add-product-button {
  display: inline-flex;
  min-height: 3rem;
  align-items: center;
  justify-content: center;
  gap: 0.55rem;
  border-radius: 0.9rem;
  padding: 0.8rem 1.15rem;
  font-size: 0.76rem;
  font-weight: 700;
  transition: 180ms ease;
}

.catalog-button {
  border: 1px solid var(--fas-border);
  background: var(--fas-card);
  color: var(--fas-text-soft);
}

.catalog-button:hover {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.add-product-button {
  background: var(--fas-accent);
  color: white;
  box-shadow: 0 14px 35px var(--fas-shadow);
}

.add-product-button:hover {
  transform: translateY(-1px);
}

.export-menu {
  position: absolute;
  right: 0;
  top: calc(100% + 0.65rem);
  z-index: 90;
  width: 270px;
  border: 1px solid var(--fas-border);
  border-radius: 1rem;
  background: var(--fas-surface);
  padding: 0.45rem;
  box-shadow: 0 24px 70px rgb(0 0 0 / 0.35);
}

.export-option {
  display: flex;
  width: 100%;
  align-items: center;
  justify-content: space-between;
  border-radius: 0.75rem;
  padding: 0.85rem 0.9rem;
  text-align: left;
}

.export-option:hover {
  background: var(--fas-hover);
}

.export-title {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--fas-text);
}

.export-subtitle {
  margin-top: 0.2rem;
  font-size: 0.62rem;
  color: var(--fas-muted);
}

.internal-option {
  margin-top: 0.25rem;
  border-top: 1px solid var(--fas-border);
}

.owner-badge {
  border-radius: 9999px;
  background: var(--fas-accent-soft);
  padding: 0.25rem 0.5rem;
  font-size: 0.52rem;
  font-weight: 800;
  color: var(--fas-accent-text);
}

.page-error,
.page-success {
  margin-top: 1.4rem;
  border-radius: 1rem;
  padding: 0.9rem 1rem;
  font-size: 0.76rem;
}

.page-error {
  border: 1px solid rgb(248 113 113 / 0.2);
  background: rgb(248 113 113 / 0.08);
  color: #f87171;
}

.page-success {
  border: 1px solid rgb(52 211 153 / 0.2);
  background: rgb(52 211 153 / 0.08);
  color: #34d399;
}

.summary-grid {
  margin-top: 2rem;
  display: grid;
  grid-template-columns:
    repeat(4, minmax(0, 1fr));
  gap: 1rem;
}

.summary-card {
  border: 1px solid var(--fas-border);
  border-radius: 1.35rem;
  background: var(--fas-card);
  padding: 1.35rem;
  box-shadow: 0 18px 45px var(--fas-shadow);
}

.summary-label {
  font-size: 0.76rem;
  color: var(--fas-muted);
}

.summary-value {
  margin-top: 0.7rem;
  font-size: 1.8rem;
  font-weight: 800;
  color: var(--fas-text);
}

.summary-footer {
  margin-top: 0.7rem;
  font-size: 0.68rem;
  color: var(--fas-muted-soft);
}

.inventory-panel {
  margin-top: 1.5rem;
  overflow: visible;
  border: 1px solid var(--fas-border);
  border-radius: 1.5rem;
  background: var(--fas-card);
  box-shadow: 0 20px 55px var(--fas-shadow);
}

.inventory-toolbar {
  display: grid;
  grid-template-columns:
    minmax(320px, 1.5fr)
    minmax(220px, 0.8fr)
    minmax(170px, 0.65fr)
    48px;
  align-items: center;
  gap: 0.9rem;
  border-bottom: 1px solid var(--fas-border);
  padding: 1.35rem;
}

.search-box {
  position: relative;
}

.search-icon {
  pointer-events: none;
  position: absolute;
  left: 1.1rem;
  top: 50%;
  z-index: 2;
  height: 1rem;
  width: 1rem;
  transform: translateY(-50%);
  color: var(--fas-muted);
}

.search-input,
.form-input {
  width: 100%;
  min-height: 3rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  color: var(--fas-text);
  outline: none;
}

.search-input {
  padding:
    0.8rem
    1rem
    0.8rem
    3.3rem;
  font-size: 0.78rem;
}

.form-input {
  padding: 0.8rem 1rem;
  font-size: 0.8rem;
}

.search-input::placeholder,
.form-input::placeholder {
  color: var(--fas-muted);
  opacity: 0.75;
}

.search-input:focus,
.form-input:focus {
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.custom-select {
  position: relative;
  min-width: 0;
}

.custom-select-trigger {
  display: flex;
  width: 100%;
  min-height: 3rem;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  padding: 0.8rem 1rem;
  font-size: 0.78rem;
  color: var(--fas-text);
}

.custom-select-trigger:hover {
  border-color: var(--fas-accent-border);
}

.custom-select-trigger:focus {
  outline: none;
}

.custom-select-trigger:focus-visible {
  outline: none;
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.chevron {
  color: var(--fas-muted);
}

.custom-select-menu {
  position: absolute;
  left: 0;
  right: 0;
  top: calc(100% + 0.45rem);
  z-index: 100;
  max-height: 15rem;
  overflow-y: auto;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-surface);
  padding: 0.4rem;
  box-shadow: var(--fas-shadow-card);
}

.custom-option {
  display: flex;
  width: 100%;
  border-radius: 0.65rem;
  padding: 0.72rem 0.8rem;
  text-align: left;
  font-size: 0.74rem;
  color: var(--fas-text-soft);
}

.custom-option:hover,
.custom-option:focus-visible,
.custom-option-active {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  outline: none;
}

.refresh-button,
.table-button,
.modal-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  color: var(--fas-muted);
}

.refresh-button {
  height: 3rem;
  width: 3rem;
}

.table-button {
  height: 2.7rem;
  width: 2.7rem;
}

.modal-close {
  height: 3rem;
  width: 3rem;
}

.refresh-button:hover,
.table-button:hover,
.modal-close:hover {
  border-color: var(--fas-accent-border);
  color: var(--fas-accent-text);
}

.table-button-toggle,
.table-button-toggle,
.table-button-danger {
  display: inline-flex;
  height: 2.7rem;
  width: 2.7rem;
  align-items: center;
  justify-content: center;
  border-radius: 0.9rem;
}

.table-button-toggle {
  border: 1px solid var(--fas-border);
  background: var(--fas-overlay);
}

.table-button-toggle.toggle-off {
  color: #f59e0b;
}

.table-button-toggle.toggle-on {
  color: #34d399;
}

.table-button-toggle:disabled {
  cursor: wait;
  opacity: 0.55;
}

.table-button-danger {
  border: 1px solid rgb(248 113 113 / 0.2);
  background: rgb(248 113 113 / 0.07);
  color: #f87171;
}

.loading-area,
.empty-area {
  display: flex;
  min-height: 18rem;
  align-items: center;
  justify-content: center;
  color: var(--fas-muted);
  font-size: 0.8rem;
}

.loading-ring {
  height: 2rem;
  width: 2rem;
  border: 3px solid var(--fas-border);
  border-top-color: var(--fas-accent);
  border-radius: 9999px;
  animation: spin 800ms linear infinite;
}

.inventory-table-scroll {
  width: 100%;
  overflow-x: auto;
  scrollbar-width: thin;
  scrollbar-color:
    var(--fas-muted-soft)
    transparent;
}

.inventory-table {
  width: 100%;
  min-width: 1700px;
  border-collapse: separate;
  border-spacing: 0;
}

.inventory-table th {
  border-right: 1px solid var(--fas-border);
  border-bottom: 1px solid var(--fas-border);
  padding: 0.95rem 1rem;
  text-align: left;
  font-size: 0.62rem;
  font-weight: 700;
  color: var(--fas-muted);
  white-space: nowrap;
}

.inventory-table td {
  border-right: 1px solid var(--fas-border);
  border-bottom: 1px solid var(--fas-border);
  padding: 1rem;
  font-size: 0.74rem;
  color: var(--fas-text-soft);
  white-space: nowrap;
}

.inventory-table th:last-child,
.inventory-table td:last-child {
  border-right: none;
}

.inventory-table tbody tr:hover {
  background: var(--fas-hover);
}

.product-cell {
  display: flex;
  min-width: 230px;
  align-items: center;
  gap: 0.8rem;
}

.product-thumbnail,
.mobile-thumbnail {
  display: flex;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px solid var(--fas-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  font-size: 0.65rem;
}

.product-thumbnail {
  height: 2.8rem;
  width: 2.8rem;
  border-radius: 0.85rem;
}

.mobile-thumbnail {
  height: 3.2rem;
  width: 3.2rem;
  border-radius: 0.9rem;
}

.product-name {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--fas-text);
}

.code-badge {
  color: var(--fas-accent-text);
  font-weight: 600;
}

.sku-badge {
  display: inline-flex;
  border-radius: 0.55rem;
  background: var(--fas-accent-soft);
  padding: 0.42rem 0.6rem;
  font-family:
    ui-monospace,
    SFMono-Regular,
    Menlo,
    monospace;
  font-size: 0.66rem;
  color: var(--fas-accent-text);
}

.barcode-value {
  font-family:
    ui-monospace,
    SFMono-Regular,
    Menlo,
    monospace;
  font-size: 0.66rem;
}

.classification-badge,
.status-badge {
  display: inline-flex;
  border-radius: 9999px;
  padding: 0.3rem 0.65rem;
  font-size: 0.56rem;
  font-weight: 800;
}

.classification-regular {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.classification-precursor {
  background: rgb(245 158 11 / 0.1);
  color: #f59e0b;
}

.status-active {
  background: rgb(52 211 153 / 0.1);
  color: #34d399;
}

.status-inactive {
  background: rgb(148 163 184 / 0.1);
  color: var(--fas-muted);
}

.price-column {
  min-width: 135px;
  text-align: right !important;
}

.selling-price {
  font-weight: 700;
  color: var(--fas-text) !important;
}

.margin-table-value {
  font-weight: 700;
  color: var(--fas-text);
}

.margin-table-rate {
  margin-top: 0.2rem;
  font-size: 0.62rem;
  color: var(--fas-accent-text);
}

.stock-value {
  font-weight: 700;
  color: var(--fas-text);
}

.stock-low {
  color: #f59e0b;
}

.stock-min {
  margin-top: 0.2rem;
  font-size: 0.62rem;
  color: var(--fas-muted);
}

.stock-status {
  display: inline-flex;
  border-radius: 9999px;
  padding: 0.32rem 0.65rem;
  font-size: 0.56rem;
  font-weight: 800;
  white-space: nowrap;
}

.stock-status-safe {
  background: rgb(52 211 153 / 0.1);
  color: #34d399;
}

.stock-status-low {
  background: rgb(245 158 11 / 0.12);
  color: #f59e0b;
}

.stock-status-empty {
  background: rgb(248 113 113 / 0.1);
  color: #f87171;
}

.action-group {
  display: flex;
  gap: 0.5rem;
}

.mobile-list {
  padding: 1rem;
}

.mobile-product {
  margin-bottom: 0.75rem;
  border: 1px solid var(--fas-border);
  border-radius: 1rem;
  background: var(--fas-overlay);
  padding: 1rem;
}

.mobile-grid {
  margin-top: 1rem;
  display: grid;
  grid-template-columns:
    repeat(2, minmax(0, 1fr));
  gap: 0.9rem;
}

.mobile-label {
  font-size: 0.62rem;
  color: var(--fas-muted);
}

.mobile-value {
  margin-top: 0.2rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--fas-text);
}

.mobile-code {
  margin-top: 0.25rem;
  color: var(--fas-accent-text);
  font-size: 0.65rem;
}

.mobile-sku {
  margin-top: 0.15rem;
  color: var(--fas-muted);
  font-size: 0.63rem;
}

.mobile-stock-limit {
  margin-top: 0.2rem;
  color: var(--fas-muted);
  font-size: 0.58rem;
}

.mobile-actions {
  margin-top: 1rem;
  display: flex;
  gap: 0.6rem;
}

.mobile-action,
.mobile-action-toggle,
.mobile-action-danger {
  flex: 1;
  border-radius: 0.75rem;
  padding: 0.7rem;
  font-size: 0.7rem;
  font-weight: 700;
}

.mobile-action {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.mobile-action-toggle {
  background: rgb(245 158 11 / 0.08);
  color: #f59e0b;
}

.mobile-action-danger {
  background: rgb(248 113 113 / 0.08);
  color: #f87171;
}

.modal-shell,
.confirm-shell {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}

.modal-shell {
  z-index: 100;
}

.confirm-shell {
  z-index: 130;
}

.modal-backdrop {
  position: absolute;
  inset: 0;
  background: rgb(0 0 0 / 0.62);
  backdrop-filter: blur(8px);
}

.product-modal {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 58rem;
  overflow: hidden;
  border: 1px solid var(--fas-border);
  border-radius: 1.5rem;
  background: var(--fas-surface);
  box-shadow: 0 35px 120px rgb(0 0 0 / 0.5);
}

.modal-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  border-bottom: 1px solid var(--fas-border);
  padding: 1.5rem 1.7rem;
}

.modal-eyebrow {
  font-size: 0.64rem;
  font-weight: 800;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--fas-accent-text);
}

.modal-title {
  margin-top: 0.45rem;
  font-size: 1.45rem;
  font-weight: 800;
  color: var(--fas-text);
}

.product-form-scroll {
  max-height: 78vh;
  overflow-y: auto;
  padding: 1.5rem 1.7rem;
  scrollbar-width: thin;
  scrollbar-color:
    var(--fas-muted-soft)
    transparent;
}

.form-error {
  margin-bottom: 1rem;
  display: flex;
  gap: 0.75rem;
  border: 1px solid rgb(248 113 113 / 0.25);
  border-radius: 0.9rem;
  background: rgb(248 113 113 / 0.08);
  padding: 0.85rem 1rem;
}

.form-error-icon {
  display: flex;
  height: 1.7rem;
  width: 1.7rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 9999px;
  background: rgb(248 113 113 / 0.15);
  font-size: 0.7rem;
  font-weight: 800;
  color: #f87171;
}

.form-error-title {
  font-size: 0.72rem;
  font-weight: 700;
  color: #f87171;
}

.form-error-text {
  margin-top: 0.2rem;
  font-size: 0.68rem;
  line-height: 1.15rem;
  color: #fca5a5;
}

.photo-section {
  display: flex;
  gap: 1rem;
  border: 1px solid var(--fas-border);
  border-radius: 1rem;
  background: var(--fas-overlay);
  padding: 1rem;
}

.photo-preview {
  display: flex;
  height: 7rem;
  width: 7rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  border: 1px dashed var(--fas-border-strong);
  border-radius: 0.9rem;
}

.photo-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.35rem;
  color: var(--fas-muted);
  font-size: 0.6rem;
}

.photo-description {
  font-size: 0.67rem;
  color: var(--fas-muted);
}

.photo-actions {
  margin-top: 0.8rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.photo-button {
  border: 1px solid var(--fas-border);
  border-radius: 0.7rem;
  background: var(--fas-card);
  padding: 0.6rem 0.8rem;
  color: var(--fas-text-soft);
  font-size: 0.68rem;
}

.photo-remove {
  border-radius: 0.7rem;
  background: rgb(248 113 113 / 0.08);
  padding: 0.6rem 0.8rem;
  color: #f87171;
  font-size: 0.68rem;
}

.form-grid {
  margin-top: 1.25rem;
  display: grid;
  grid-template-columns:
    repeat(2, minmax(0, 1fr));
  gap: 1rem;
}

.span-two {
  grid-column: span 2;
}

.form-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.form-label {
  margin-bottom: 0.48rem;
  display: block;
  font-size: 0.7rem;
  font-weight: 700;
  color: var(--fas-text-soft);
}

.optional-label {
  font-size: 0.55rem;
  font-weight: 700;
  color: var(--fas-muted);
}

.generated-field {
  display: flex;
  min-height: 3rem;
  align-items: center;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  padding: 0.65rem 1rem;
}

.generated-value {
  font-size: 0.78rem;
  font-weight: 700;
  color: var(--fas-text);
}

.generated-sub {
  margin-top: 0.15rem;
  font-size: 0.58rem;
  color: var(--fas-muted);
}

.sku-auto-button {
  margin-bottom: 0.42rem;
  border-radius: 0.5rem;
  background: var(--fas-accent-soft);
  padding: 0.28rem 0.5rem;
  color: var(--fas-accent-text);
  font-size: 0.55rem;
  font-weight: 800;
}

.measurement-row {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    130px;
  gap: 0.65rem;
}

.quick-size-list {
  margin-top: 0.6rem;
  display: flex;
  flex-wrap: wrap;
  gap: 0.38rem;
}

.quick-size {
  min-width: 2.6rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.55rem;
  background: var(--fas-overlay);
  padding: 0.36rem 0.5rem;
  font-size: 0.58rem;
  font-weight: 600;
  color: var(--fas-muted);
}

.quick-size:hover {
  border-color: var(--fas-accent-border);
  color: var(--fas-accent-text);
}

.quick-size-active {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.field-helper {
  margin-top: 0.5rem;
  font-size: 0.6rem;
  color: var(--fas-muted);
}

.margin-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  padding: 1rem;
}

.margin-label {
  font-size: 0.64rem;
  color: var(--fas-muted);
}

.margin-value,
.margin-percent {
  margin-top: 0.2rem;
  font-size: 0.9rem;
  font-weight: 800;
}

.margin-value {
  color: var(--fas-text);
}

.margin-percent {
  color: var(--fas-accent-text);
}

.margin-negative {
  color: #f87171;
}

.form-footer {
  margin-top: 1.5rem;
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  border-top: 1px solid var(--fas-border);
  padding-top: 1rem;
}

.primary-button,
.secondary-button {
  display: inline-flex;
  min-height: 2.9rem;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 0.8rem;
  padding: 0.75rem 1.1rem;
  font-size: 0.72rem;
  font-weight: 700;
}

.primary-button {
  background: var(--fas-accent);
  color: white;
}

.secondary-button {
  border: 1px solid var(--fas-border);
  background: var(--fas-overlay);
  color: var(--fas-muted);
}

.primary-button:disabled,
.secondary-button:disabled {
  cursor: not-allowed;
  opacity: 0.65;
}

.button-spinner {
  height: 0.9rem;
  width: 0.9rem;
  border: 2px solid rgb(255 255 255 / 0.3);
  border-top-color: currentColor;
  border-radius: 9999px;
  animation: spin 700ms linear infinite;
}

.confirm-dialog {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 430px;
  border: 1px solid var(--fas-border);
  border-radius: 1.5rem;
  background: var(--fas-surface);
  padding: 1.5rem;
  box-shadow: 0 40px 120px rgb(0 0 0 / 0.5);
}

.confirm-icon {
  display: flex;
  height: 3.4rem;
  width: 3.4rem;
  align-items: center;
  justify-content: center;
  border: 1px solid rgb(248 113 113 / 0.2);
  border-radius: 1rem;
  background: rgb(248 113 113 / 0.08);
  color: #f87171;
}

.confirm-eyebrow {
  margin-top: 1.2rem;
  font-size: 0.6rem;
  font-weight: 800;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #f87171;
}

.confirm-title {
  margin-top: 0.45rem;
  font-size: 1.3rem;
  font-weight: 800;
  color: var(--fas-text);
}

.confirm-description {
  margin-top: 0.7rem;
  font-size: 0.75rem;
  line-height: 1.35rem;
  color: var(--fas-muted);
}

.confirm-description strong {
  color: var(--fas-text);
}

.confirm-product {
  margin-top: 1rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-overlay);
  padding: 0.9rem 1rem;
}

.confirm-product-name {
  font-size: 0.76rem;
  font-weight: 700;
  color: var(--fas-text);
}

.confirm-product-code {
  margin-top: 0.2rem;
  font-size: 0.62rem;
  color: var(--fas-muted);
}

.confirm-status {
  border-radius: 9999px;
  background: rgb(52 211 153 / 0.1);
  padding: 0.3rem 0.55rem;
  font-size: 0.54rem;
  font-weight: 800;
  color: #34d399;
}

.confirm-note {
  margin-top: 0.9rem;
  font-size: 0.65rem;
  line-height: 1.15rem;
  color: var(--fas-muted);
}

.confirm-actions {
  margin-top: 1.4rem;
  display: grid;
  grid-template-columns:
    1fr 1fr;
  gap: 0.7rem;
}

.confirm-cancel,
.confirm-danger {
  display: inline-flex;
  min-height: 2.9rem;
  align-items: center;
  justify-content: center;
  gap: 0.4rem;
  border-radius: 0.8rem;
  padding: 0.75rem 1rem;
  font-size: 0.72rem;
  font-weight: 700;
}

.confirm-cancel {
  border: 1px solid var(--fas-border);
  background: var(--fas-overlay);
  color: var(--fas-text-soft);
}

.confirm-danger {
  background: #dc2626;
  color: white;
}

.dropdown-enter-active,
.dropdown-leave-active {
  transition:
    opacity 140ms ease,
    transform 140ms ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-5px);
}

.modal-enter-active,
.modal-leave-active,
.confirm-enter-active,
.confirm-leave-active {
  transition: opacity 180ms ease;
}

.modal-enter-from,
.modal-leave-to,
.confirm-enter-from,
.confirm-leave-to {
  opacity: 0;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .summary-grid {
    grid-template-columns:
      repeat(2, minmax(0, 1fr));
  }

  .inventory-toolbar {
    grid-template-columns:
      1fr 1fr;
  }

  .search-box {
    grid-column: 1 / -1;
  }
}

@media (max-width: 800px) {
  .inventory-heading {
    flex-direction: column;
    align-items: stretch;
  }

  .heading-actions {
    width: 100%;
  }

  .heading-actions > * {
    flex: 1;
  }

  .catalog-button,
  .add-product-button {
    width: 100%;
  }
}

@media (max-width: 639px) {
  .inventory-title {
    font-size: 1.7rem;
  }

  .heading-actions {
    flex-direction: column-reverse;
  }

  .export-wrapper {
    width: 100%;
  }

  .export-menu {
    left: 0;
    right: 0;
    width: auto;
  }

  .summary-grid {
    grid-template-columns: 1fr 1fr;
  }

  .inventory-toolbar {
    grid-template-columns: 1fr;
    padding: 1rem;
  }

  .search-box {
    grid-column: auto;
  }

  .refresh-button {
    width: 100%;
  }

  .form-grid {
    grid-template-columns: 1fr;
  }

  .span-two {
    grid-column: auto;
  }

  .measurement-row {
    grid-template-columns:
      minmax(0, 1fr)
      105px;
  }

  .photo-section {
    flex-direction: column;
  }

  .photo-preview {
    width: 100%;
    height: 11rem;
  }

  .form-footer {
    flex-direction: column-reverse;
  }

  .primary-button,
  .secondary-button {
    width: 100%;
  }

  .confirm-actions {
    grid-template-columns: 1fr;
  }
}

.chevron {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.chevron svg {
  width: 1.125rem;
  height: 1.125rem;
}

.custom-select-trigger {
  border-color: var(--fas-border-strong);
  background: var(--fas-input);
}

.custom-select-menu {
  background: var(--fas-elevated);
  border-color: var(--fas-accent-border);
}

.manual-warehouse {
  display: flex;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 0.65rem;
  margin-top: 0.8rem;
}
.manual-warehouse label { min-width: 0; flex: 1; }
.manual-warehouse label > span { display: block; margin-bottom: 0.35rem; color: var(--fas-muted); font-size: 0.7rem; }
.warehouse-add-button { min-height: 3rem; border: 1px solid var(--fas-accent-border); border-radius: 0.85rem; background: var(--fas-accent-soft); color: var(--fas-accent-text); padding: 0.6rem 0.85rem; font-size: 0.75rem; cursor: pointer; }
.warehouse-add-button:disabled { opacity: 0.55; cursor: not-allowed; }
</style>