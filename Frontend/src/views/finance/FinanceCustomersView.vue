<script setup>
import FinanceNavigation from '@/components/finance/FinanceNavigation.vue'
import { companyProfile, companyDocumentHTML, loadCompanyProfile } from '@/services/company'

import FasIcon from '@/components/ui/FasIcon.vue'
import { useModalDialog } from '@/composables/useModalDialog'
import FasSelect from '@/components/ui/FasSelect.vue'

import {
  computed,
  onBeforeUnmount,
  onMounted,
  reactive,
  ref,
  watch,
} from 'vue'

import {
  getFinanceInvoiceByID,
  getFinanceSales,
  getFinanceSalesSummary,
} from '@/services/finance'

import {
  markInvoicePrinted,
} from '@/services/invoice'

import {
  createCustomer,
  getCustomers,
  updateCustomer,
} from '@/services/customer'

const AUTO_REFRESH_MS = 15000
const PAGE_LIMIT = 50
const EXPORT_BATCH_LIMIT = 200

const loading = ref(false)
const refreshing = ref(false)
const exporting = ref(false)
const invoiceLoading = ref(false)
const invoicePrinting = ref(false)

const errorMessage = ref('')
const successMessage = ref('')
const lastUpdatedAt = ref(null)

const customerLoading = ref(false)
const customerSaving = ref(false)
const customerExporting = ref(false)
const customers = ref([])
const customerSearch = ref('')
const customerTaxFilter = ref('ALL')
const customerStatusFilter = ref('ACTIVE')
const customerModalOpen = ref(false)
const customerModalMode = ref('edit')
const selectedCustomer = ref(null)

const customerForm = reactive({
  name: '',
  phone: '',
  email: '',
  shipping_address: '',
  tax_id_type: '',
  tax_id_number: '',
  tax_name: '',
  tax_address: '',
  is_pkp: false,
  tku_id: '',
  payment_terms_days: 0,
  is_active: true,
})

const sales = ref([])
const meta = ref({
  date_from: '',
  date_to: '',
  query: '',
  status: '',
  print_state: 'all',
  limit: PAGE_LIMIT,
  offset: 0,
  total: 0,
})

const summary = ref({
  invoice_count: 0,
  customer_count: 0,
  total_qty: 0,
  subtotal: 0,
  discount_amount: 0,
  net_sales: 0,
  tax_base_amount: 0,
  tax_amount: 0,
  grand_total: 0,
  average_invoice: 0,
  printed_count: 0,
  unprinted_count: 0,
  term_count: 0,
  immediate_count: 0,
})

const filters = reactive({
  query: '',
  date_from: '',
  date_to: '',
  print_state: 'all',
})

const activePreset = ref('THIS_MONTH')

const reportPreviewOpen = ref(false)
const reportPreviewLoading = ref(false)
const reportPreviewRows = ref([])

const invoiceOpen = ref(false)
const selectedInvoice = ref(null)

const RECIPIENT_STORAGE_KEY =
  'fas_distribution_invoice_recipients'

let refreshTimer = null
let searchTimer = null

const todayISO = () => {
  const date = new Date()
  const year = date.getFullYear()
  const month = String(
    date.getMonth() + 1,
  ).padStart(2, '0')
  const day = String(
    date.getDate(),
  ).padStart(2, '0')

  return `${year}-${month}-${day}`
}

const toISODate = date => {
  const year = date.getFullYear()
  const month = String(
    date.getMonth() + 1,
  ).padStart(2, '0')
  const day = String(
    date.getDate(),
  ).padStart(2, '0')

  return `${year}-${month}-${day}`
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

const firstDayOfYearISO = () => {
  const now = new Date()

  return toISODate(
    new Date(
      now.getFullYear(),
      0,
      1,
    ),
  )
}

const sixMonthsAgoISO = () => {
  const now = new Date()

  return toISODate(
    new Date(
      now.getFullYear(),
      now.getMonth() - 5,
      1,
    ),
  )
}

const setInitialPeriod = () => {
  filters.date_from =
    firstDayOfMonthISO()

  filters.date_to =
    todayISO()
}

setInitialPeriod()

const formatCurrency = value => {
  const number = Number(value || 0)

  return new Intl.NumberFormat(
    'id-ID',
    {
      style: 'currency',
      currency: 'IDR',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    },
  ).format(number)
}

const formatCompactCurrency = value => {
  const number = Number(value || 0)

  return new Intl.NumberFormat(
    'id-ID',
    {
      style: 'currency',
      currency: 'IDR',
      notation: 'compact',
      compactDisplay: 'short',
      minimumFractionDigits: 0,
      maximumFractionDigits: 1,
    },
  ).format(number)
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

  const parsed = new Date(value)

  if (
    Number.isNaN(
      parsed.getTime(),
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
  ).format(parsed)
}

const formatDateTime = value => {
  if (!value) {
    return '—'
  }

  const parsed = new Date(value)

  if (
    Number.isNaN(
      parsed.getTime(),
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
    },
  ).format(parsed)
}

const escapeHTML = value => {
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

const normalizeString = value => {
  return String(
    value ?? '',
  ).trim()
}


const normalizeCustomerTaxNumber = value => {
  return String(
    value ?? '',
  )
    .replaceAll('.', '')
    .replaceAll('-', '')
    .replaceAll(' ', '')
    .trim()
}


const normalizeCustomerPhone = value => {
  return String(
    value ?? '',
  )
    .replace(/\D/g, '')
    .trim()
}

const normalizeCustomerName = value => {
  return String(
    value ?? '',
  )
    .trim()
    .toLowerCase()
    .replace(/\s+/g, ' ')
}

const customerDuplicateCandidates = computed(() => {
  if (
    customerModalMode.value !==
    'create'
  ) {
    return []
  }

  const taxNumber =
    normalizeCustomerTaxNumber(
      customerForm.tax_id_number,
    )

  const phone =
    normalizeCustomerPhone(
      customerForm.phone,
    )

  const name =
    normalizeCustomerName(
      customerForm.name,
    )

  if (
    !taxNumber &&
    !phone &&
    !name
  ) {
    return []
  }

  return customers.value
    .filter(customer => {
      const customerTax =
        normalizeCustomerTaxNumber(
          customer.tax_id_number,
        )

      const customerPhone =
        normalizeCustomerPhone(
          customer.phone,
        )

      const customerName =
        normalizeCustomerName(
          customer.name,
        )

      const sameTax =
        Boolean(
          taxNumber &&
          customerTax &&
          taxNumber ===
            customerTax,
        )

      const samePhone =
        Boolean(
          phone &&
          customerPhone &&
          phone ===
            customerPhone,
        )

      const sameName =
        Boolean(
          name &&
          customerName &&
          name ===
            customerName,
        )

      return (
        sameTax ||
        samePhone ||
        sameName
      )
    })
    .slice(0, 5)
})

const blockingDuplicateCustomer = computed(() => {
  if (
    customerModalMode.value !==
    'create'
  ) {
    return null
  }

  const taxNumber =
    normalizeCustomerTaxNumber(
      customerForm.tax_id_number,
    )

  const phone =
    normalizeCustomerPhone(
      customerForm.phone,
    )

  return (
    customers.value.find(
      customer => {
        const sameTax =
          Boolean(
            taxNumber &&
            normalizeCustomerTaxNumber(
              customer.tax_id_number,
            ) ===
              taxNumber,
          )

        const samePhone =
          Boolean(
            phone &&
            normalizeCustomerPhone(
              customer.phone,
            ) ===
              phone,
          )

        return (
          sameTax ||
          samePhone
        )
      },
    ) ||
    null
  )
})

const customerTaxComplete = customer => {
  return Boolean(
    String(
      customer?.tax_id_type ||
      '',
    ).trim() &&
    String(
      customer?.tax_id_number ||
      '',
    ).trim() &&
    String(
      customer?.tax_name ||
      '',
    ).trim() &&
    String(
      customer?.tax_address ||
      '',
    ).trim(),
  )
}

const customerTaxStatusLabel = customer => {
  return customerTaxComplete(customer)
    ? 'Data Pajak Lengkap'
    : 'Perlu Dilengkapi'
}

const customerTaxIdentityLabel = customer => {
  const type =
    String(
      customer?.tax_id_type ||
      '',
    ).trim()

  const number =
    String(
      customer?.tax_id_number ||
      '',
    ).trim()

  if (!type && !number) {
    return 'Belum ada'
  }

  return [
    type,
    number,
  ]
    .filter(Boolean)
    .join(' · ')
}

const filteredCustomers = computed(() => {
  const query =
    customerSearch.value
      .trim()
      .toLowerCase()

  return customers.value
    .filter(customer => {
      if (
        customerStatusFilter.value ===
          'ACTIVE' &&
        !customer.is_active
      ) {
        return false
      }

      if (
        customerStatusFilter.value ===
          'INACTIVE' &&
        customer.is_active
      ) {
        return false
      }

      const complete =
        customerTaxComplete(
          customer,
        )

      if (
        customerTaxFilter.value ===
          'COMPLETE' &&
        !complete
      ) {
        return false
      }

      if (
        customerTaxFilter.value ===
          'INCOMPLETE' &&
        complete
      ) {
        return false
      }

      if (
        customerTaxFilter.value ===
          'PKP' &&
        !customer.is_pkp
      ) {
        return false
      }

      if (
        customerTaxFilter.value ===
          'NON_PKP' &&
        customer.is_pkp
      ) {
        return false
      }

      if (!query) {
        return true
      }

      const searchable =
        [
          customer.customer_code,
          customer.name,
          customer.phone,
          customer.email,
          customer.shipping_address,
          customer.tax_id_type,
          customer.tax_id_number,
          customer.tax_name,
          customer.tax_address,
          customer.tku_id,
        ]
          .map(value =>
            String(
              value ??
              '',
            ).toLowerCase(),
          )
          .join(' ')

      return searchable.includes(
        query,
      )
    })
    .sort(
      (a, b) =>
        String(
          a?.name ||
          '',
        ).localeCompare(
          String(
            b?.name ||
            '',
          ),
          'id',
        ),
    )
})

const customerSummary = computed(() => {
  const active =
    customers.value.filter(
      customer =>
        customer.is_active,
    )

  const complete =
    active.filter(
      customer =>
        customerTaxComplete(
          customer,
        ),
    )

  const incomplete =
    active.filter(
      customer =>
        !customerTaxComplete(
          customer,
        ),
    )

  const pkp =
    active.filter(
      customer =>
        customer.is_pkp,
    )

  return {
    total:
      customers.value.length,
    active:
      active.length,
    complete:
      complete.length,
    incomplete:
      incomplete.length,
    pkp:
      pkp.length,
  }
})

const loadCustomers = async ({
  silent = false,
} = {}) => {
  if (!silent) {
    customerLoading.value =
      true
  }

  try {
    const response =
      await getCustomers()

    customers.value =
      Array.isArray(response)
        ? response
        : []
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Data customer belum dapat dimuat.'
  } finally {
    customerLoading.value =
      false
  }
}

const resetCustomerForm = () => {
  customerForm.name = ''
  customerForm.phone = ''
  customerForm.email = ''
  customerForm.shipping_address = ''
  customerForm.tax_id_type = ''
  customerForm.tax_id_number = ''
  customerForm.tax_name = ''
  customerForm.tax_address = ''
  customerForm.is_pkp = false
  customerForm.tku_id = ''
  customerForm.payment_terms_days = 0
  customerForm.is_active = true
}


const openCreateCustomer = () => {
  resetCustomerForm()

  customerModalMode.value =
    'create'

  selectedCustomer.value =
    null

  customerForm.is_active =
    true

  errorMessage.value = ''
  successMessage.value = ''

  customerModalOpen.value =
    true
}

const openCustomerEditor = customer => {
  customerModalMode.value =
    'edit'

  selectedCustomer.value =
    customer

  customerForm.name =
    customer?.name ||
    ''

  customerForm.phone =
    customer?.phone ||
    ''

  customerForm.email =
    customer?.email ||
    ''

  customerForm.shipping_address =
    customer?.shipping_address ||
    ''

  customerForm.tax_id_type =
    customer?.tax_id_type ||
    ''

  customerForm.tax_id_number =
    customer?.tax_id_number ||
    ''

  customerForm.tax_name =
    customer?.tax_name ||
    ''

  customerForm.tax_address =
    customer?.tax_address ||
    ''

  customerForm.is_pkp =
    Boolean(
      customer?.is_pkp,
    )

  customerForm.tku_id =
    customer?.tku_id ||
    ''

  customerForm.payment_terms_days =
    Number(
      customer?.payment_terms_days ||
      0,
    )

  customerForm.is_active =
    customer?.is_active !==
    false

  errorMessage.value = ''
  successMessage.value = ''

  customerModalOpen.value =
    true
}

const closeCustomerEditor = () => {
  if (customerSaving.value) {
    return
  }

  customerModalOpen.value =
    false

  selectedCustomer.value =
    null

  customerModalMode.value =
    'edit'

  resetCustomerForm()
}

const validateCustomerForm = () => {
  if (
    !customerForm.name.trim()
  ) {
    return 'Nama toko wajib diisi.'
  }

  const type =
    customerForm.tax_id_type

  if (
    type !== '' &&
    type !== 'NPWP' &&
    type !== 'NIK'
  ) {
    return 'Jenis identitas pajak harus NPWP, NIK, atau dikosongkan.'
  }

  if (
    Number(
      customerForm.payment_terms_days,
    ) < 0
  ) {
    return 'Tempo pembayaran tidak boleh negatif.'
  }

  return ''
}

const saveCustomer = async () => {
  if (customerSaving.value) {
    return
  }

  if (
    customerModalMode.value ===
      'edit' &&
    !selectedCustomer.value?.id
  ) {
    return
  }

  const validation =
    validateCustomerForm()

  if (validation) {
    errorMessage.value =
      validation
    return
  }

  if (
    customerModalMode.value ===
      'create' &&
    blockingDuplicateCustomer.value
  ) {
    errorMessage.value =
      `Customer kemungkinan sudah terdaftar sebagai ${blockingDuplicateCustomer.value.name} (${blockingDuplicateCustomer.value.customer_code}). Buka data customer tersebut lalu gunakan Edit agar tidak membuat duplikat.`

    return
  }

  customerSaving.value =
    true

  errorMessage.value = ''
  successMessage.value = ''

  const payload = {
    name:
      customerForm.name.trim(),

    phone:
      customerForm.phone.trim(),

    email:
      customerForm.email
        .trim()
        .toLowerCase(),

    shipping_address:
      customerForm
        .shipping_address
        .trim(),

    tax_id_type:
      customerForm
        .tax_id_type,

    tax_id_number:
      normalizeCustomerTaxNumber(
        customerForm
          .tax_id_number,
      ),

    tax_name:
      customerForm
        .tax_name
        .trim(),

    tax_address:
      customerForm
        .tax_address
        .trim(),

    is_pkp:
      Boolean(
        customerForm.is_pkp,
      ),

    tku_id:
      customerForm
        .tku_id
        .trim(),

    payment_terms_days:
      Number(
        customerForm
          .payment_terms_days ||
        0,
      ),

    is_active:
      Boolean(
        customerForm.is_active,
      ),
  }

  try {
    if (
      customerModalMode.value ===
      'create'
    ) {
      const created =
        await createCustomer(
          payload,
        )

      await loadCustomers({
        silent: true,
      })

      successMessage.value =
        `Toko ${created.name} berhasil ditambahkan ke Customer Master. Data ini sekarang tersedia untuk Finance, Operations, Distribution, faktur, piutang, dan laporan.`

      customerSearch.value =
        created.name || ''

      customerModalOpen.value =
        false

      selectedCustomer.value =
        null

      customerModalMode.value =
        'edit'

      resetCustomerForm()

      return
    }

    const updated =
      await updateCustomer(
        selectedCustomer.value.id,
        payload,
      )

    const index =
      customers.value.findIndex(
        customer =>
          customer.id ===
          updated.id,
      )

    if (index >= 0) {
      customers.value[
        index
      ] = updated
    }

    selectedCustomer.value =
      updated

    successMessage.value =
      `Data customer ${updated.name} berhasil diperbarui dan tersinkron ke Customer Master.`

    customerModalOpen.value =
      false

    selectedCustomer.value =
      null

    customerModalMode.value =
      'edit'

    resetCustomerForm()
  } catch (error) {
    const status =
      error.response?.status

    const backendMessage =
      error.response?.data?.error

    if (status === 409) {
      errorMessage.value =
        backendMessage ||
        'Customer dengan data yang sama sudah terdaftar. Cari customer tersebut lalu gunakan Edit.'
    } else {
      errorMessage.value =
        backendMessage ||
        (
          customerModalMode.value ===
          'create'
            ? 'Toko baru belum dapat disimpan.'
            : 'Data customer belum dapat disimpan.'
        )
    }
  } finally {
    customerSaving.value =
      false
  }
}

const buildCustomerReportRows = rows => {
  return rows
    .map(
      (
        customer,
        index,
      ) => `
        <tr>
          <td>${index + 1}</td>
          <td>${escapeHTML(customer.customer_code || '-')}</td>
          <td>${escapeHTML(customer.name || '-')}</td>
          <td>${escapeHTML(customer.phone || '-')}</td>
          <td>${escapeHTML(customer.email || '-')}</td>
          <td>${escapeHTML(customer.shipping_address || '-')}</td>
          <td>${escapeHTML(customer.tax_id_type || '-')}</td>
          <td>${escapeHTML(customer.tax_id_number || '-')}</td>
          <td>${escapeHTML(customer.tax_name || '-')}</td>
          <td>${escapeHTML(customer.tax_address || '-')}</td>
          <td>${customer.is_pkp ? 'PKP' : 'Non-PKP'}</td>
          <td>${escapeHTML(customer.tku_id || '-')}</td>
          <td>${Number(customer.payment_terms_days || 0)} hari</td>
          <td>${customer.is_active ? 'Aktif' : 'Nonaktif'}</td>
        </tr>
      `,
    )
    .join('')
}

const printCustomerReport = () => {
  customerExporting.value =
    true

  errorMessage.value = ''

  try {
    const rows =
      filteredCustomers.value

    const win =
      window.open(
        '',
        '_blank',
      )

    if (!win) {
      throw new Error(
        'Pop-up browser diblokir. Izinkan pop-up untuk mencetak data customer.',
      )
    }

    win.opener = null
    win.document.open()

    win.document.write(`
      <!DOCTYPE html>
      <html lang="id">
      <head>
        <meta charset="UTF-8" />
        <title>Data Customer & Pajak PT FAS</title>

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
            font-family:
              Arial,
              Helvetica,
              sans-serif;
            color: #111827;
          }

          .header {
            display: flex;
            align-items: flex-start;
            justify-content: space-between;
            gap: 24px;
            border-bottom: 2px solid #111827;
            padding-bottom: 10px;
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
            font-size: 9px;
            line-height: 1.5;
            text-align: right;
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
            font-size: 8px;
          }

          .card strong {
            display: block;
            margin-top: 3px;
            font-size: 11px;
          }

          table {
            width: 100%;
            border-collapse: collapse;
          }

          th,
          td {
            border: 1px solid #94a3b8;
            padding: 4px 5px;
            font-size: 7px;
            vertical-align: top;
          }

          th {
            background: #f1f5f9;
            font-weight: 700;
            text-align: left;
          }
        </style>
      </head>

      <body>${companyDocumentHTML()}
        <div class="header">
          <div>
            <h1>
              ${escapeHTML(companyProfile.name)}
            </h1>

            <h2>
              Data Customer & Pajak
            </h2>
          </div>

          <div class="meta">
            Dicetak:
            ${escapeHTML(
              formatDateTime(
                new Date(),
              ),
            )}
            <br />
            Hasil filter:
            ${escapeHTML(
              formatNumber(
                rows.length,
              ),
            )}
            customer
          </div>
        </div>

        <section class="summary">
          <div class="card">
            <span>Customer Aktif</span>
            <strong>
              ${escapeHTML(
                formatNumber(
                  customerSummary
                    .value.active,
                ),
              )}
            </strong>
          </div>

          <div class="card">
            <span>Data Pajak Lengkap</span>
            <strong>
              ${escapeHTML(
                formatNumber(
                  customerSummary
                    .value.complete,
                ),
              )}
            </strong>
          </div>

          <div class="card">
            <span>Perlu Dilengkapi</span>
            <strong>
              ${escapeHTML(
                formatNumber(
                  customerSummary
                    .value.incomplete,
                ),
              )}
            </strong>
          </div>

          <div class="card">
            <span>PKP</span>
            <strong>
              ${escapeHTML(
                formatNumber(
                  customerSummary
                    .value.pkp,
                ),
              )}
            </strong>
          </div>
        </section>

        <table>
          <thead>
            <tr>
              <th>No</th>
              <th>Kode</th>
              <th>Nama Toko</th>
              <th>Telepon</th>
              <th>Email</th>
              <th>Alamat</th>
              <th>ID Pajak</th>
              <th>NPWP / NIK</th>
              <th>Nama Pajak</th>
              <th>Alamat Pajak</th>
              <th>PKP</th>
              <th>TKU</th>
              <th>Tempo</th>
              <th>Status</th>
            </tr>
          </thead>

          <tbody>
            ${buildCustomerReportRows(
              rows,
            )}
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
    errorMessage.value =
      error.message ||
      'Data customer belum dapat dicetak.'
  } finally {
    customerExporting.value =
      false
  }
}

const exportCustomersExcel = () => {
  customerExporting.value =
    true

  errorMessage.value = ''

  try {
    const rows =
      filteredCustomers.value

    const body =
      rows
        .map(
          (
            customer,
            index,
          ) => `
            <tr>
              <td>${index + 1}</td>
              <td>${excelCell(customer.customer_code)}</td>
              <td>${excelCell(customer.name)}</td>
              <td>${excelCell(customer.phone)}</td>
              <td>${excelCell(customer.email)}</td>
              <td>${excelCell(customer.shipping_address)}</td>
              <td>${excelCell(customer.tax_id_type)}</td>
              <td>${excelCell(customer.tax_id_number)}</td>
              <td>${excelCell(customer.tax_name)}</td>
              <td>${excelCell(customer.tax_address)}</td>
              <td>${customer.is_pkp ? 'PKP' : 'Non-PKP'}</td>
              <td>${excelCell(customer.tku_id)}</td>
              <td>${Number(customer.payment_terms_days || 0)}</td>
              <td>${customer.is_active ? 'Aktif' : 'Nonaktif'}</td>
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
            <th colspan="14">
              ${escapeHTML(companyProfile.name)} - DATA CUSTOMER & PAJAK
            </th>
          </tr>

          <tr>
            <td colspan="14">
              Diekspor ${excelCell(formatDateTime(new Date()))}
            </td>
          </tr>

          <tr>
            <th>No</th>
            <th>Kode Customer</th>
            <th>Nama Toko</th>
            <th>Telepon</th>
            <th>Email</th>
            <th>Alamat Pengiriman</th>
            <th>Jenis ID Pajak</th>
            <th>NPWP / NIK</th>
            <th>Nama Pajak</th>
            <th>Alamat Pajak</th>
            <th>PKP</th>
            <th>TKU</th>
            <th>Tempo Default</th>
            <th>Status</th>
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
      'customer-pajak-pt-fas.xls'

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
      'Data customer belum dapat diekspor.'
  } finally {
    customerExporting.value =
      false
  }
}

const requestParams = computed(() => {
  return {
    date_from:
      filters.date_from,
    date_to:
      filters.date_to,
    q:
      filters.query.trim(),
    print_state:
      filters.print_state,
    limit:
      PAGE_LIMIT,
    offset:
      meta.value.offset || 0,
  }
})

const totalPages = computed(() => {
  const total =
    Number(
      meta.value.total || 0,
    )

  if (total <= 0) {
    return 1
  }

  return Math.max(
    1,
    Math.ceil(
      total /
      PAGE_LIMIT,
    ),
  )
})

const currentPage = computed(() => {
  return (
    Math.floor(
      Number(
        meta.value.offset || 0,
      ) /
      PAGE_LIMIT,
    ) +
    1
  )
})

const canGoPrevious = computed(() => {
  return (
    Number(
      meta.value.offset || 0,
    ) > 0
  )
})

const canGoNext = computed(() => {
  return (
    Number(
      meta.value.offset || 0,
    ) +
      PAGE_LIMIT <
    Number(
      meta.value.total || 0,
    )
  )
})

const termRatio = computed(() => {
  const total =
    Number(
      summary.value.invoice_count ||
      0,
    )

  if (total <= 0) {
    return 0
  }

  return Math.round(
    (
      Number(
        summary.value.term_count ||
        0,
      ) /
      total
    ) *
      100,
  )
})

const immediateRatio = computed(() => {
  const total =
    Number(
      summary.value.invoice_count ||
      0,
    )

  if (total <= 0) {
    return 0
  }

  return Math.round(
    (
      Number(
        summary.value.immediate_count ||
        0,
      ) /
      total
    ) *
      100,
  )
})

const printedRatio = computed(() => {
  const total =
    Number(
      summary.value.invoice_count ||
      0,
    )

  if (total <= 0) {
    return 0
  }

  return Math.round(
    (
      Number(
        summary.value.printed_count ||
        0,
      ) /
      total
    ) *
      100,
  )
})

const periodLabel = computed(() => {
  if (
    !filters.date_from ||
    !filters.date_to
  ) {
    return 'Semua periode'
  }

  return (
    `${formatDate(filters.date_from)} ` +
    `— ${formatDate(filters.date_to)}`
  )
})

const realTimeLabel = computed(() => {
  if (!lastUpdatedAt.value) {
    return 'Menunggu sinkronisasi'
  }

  return (
    `Diperbarui ` +
    formatDateTime(
      lastUpdatedAt.value,
    )
  )
})

const buildBaseParams = () => {
  return {
    date_from:
      filters.date_from,
    date_to:
      filters.date_to,
    q:
      filters.query.trim(),
    print_state:
      filters.print_state,
  }
}

const loadFinanceSales = async ({
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
      ...buildBaseParams(),
      limit:
        PAGE_LIMIT,
      offset:
        Number(
          meta.value.offset ||
          0,
        ),
    }

    const [
      salesResponse,
      summaryResponse,
    ] =
      await Promise.all([
        getFinanceSales(
          params,
        ),
        getFinanceSalesSummary(
          buildBaseParams(),
        ),
      ])

    sales.value =
      Array.isArray(
        salesResponse?.data,
      )
        ? salesResponse.data
        : []

    meta.value = {
      ...meta.value,
      ...(
        salesResponse?.meta ||
        {}
      ),
    }

    summary.value = {
      ...summary.value,
      ...(
        summaryResponse?.data ||
        {}
      ),
    }

    lastUpdatedAt.value =
      new Date()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      'Data Finance belum dapat dimuat.'
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

const resetPagination = () => {
  meta.value.offset = 0
}

const refreshNow = async () => {
  await Promise.all([
    loadFinanceSales({
      silent: true,
    }),
    loadCustomers({
      silent: true,
    }),
  ])
}

const applyPreset = async preset => {
  activePreset.value =
    preset

  const now =
    new Date()

  if (preset === 'TODAY') {
    filters.date_from =
      todayISO()
    filters.date_to =
      todayISO()
  }

  if (preset === 'THIS_MONTH') {
    filters.date_from =
      firstDayOfMonthISO()
    filters.date_to =
      todayISO()
  }

  if (preset === 'SIX_MONTHS') {
    filters.date_from =
      sixMonthsAgoISO()
    filters.date_to =
      todayISO()
  }

  if (preset === 'THIS_YEAR') {
    filters.date_from =
      firstDayOfYearISO()
    filters.date_to =
      todayISO()
  }

  if (preset === 'LAST_30_DAYS') {
    const start =
      new Date(
        now.getFullYear(),
        now.getMonth(),
        now.getDate() - 29,
      )

    filters.date_from =
      toISODate(start)
    filters.date_to =
      todayISO()
  }

  resetPagination()

  await loadFinanceSales()
}

const useCustomPeriod = async () => {
  activePreset.value =
    'CUSTOM'

  resetPagination()

  await loadFinanceSales()
}

const clearSearch = async () => {
  filters.query = ''

  resetPagination()

  await loadFinanceSales()
}

const handlePrintStateChange = async () => {
  resetPagination()

  await loadFinanceSales()
}

const previousPage = async () => {
  if (!canGoPrevious.value) {
    return
  }

  meta.value.offset =
    Math.max(
      0,
      Number(
        meta.value.offset ||
        0,
      ) -
        PAGE_LIMIT,
    )

  await loadFinanceSales()
}

const nextPage = async () => {
  if (!canGoNext.value) {
    return
  }

  meta.value.offset =
    Number(
      meta.value.offset ||
      0,
    ) +
    PAGE_LIMIT

  await loadFinanceSales()
}

const paymentSettlementLabel = value => {
  const normalized =
    normalizeString(value)
      .toUpperCase()

  if (normalized === 'TERM') {
    return 'Tempo'
  }

  if (
    normalized === 'IMMEDIATE'
  ) {
    return 'Langsung'
  }

  return (
    normalized ||
    'Belum ditentukan'
  )
}

const paymentChannelLabel = sale => {
  const channel =
    normalizeString(
      sale?.payment_channel,
    )

  const provider =
    normalizeString(
      sale?.payment_provider,
    )

  if (
    !channel ||
    channel === 'UNSPECIFIED'
  ) {
    return 'Belum ditentukan'
  }

  if (provider) {
    return `${channel} · ${provider}`
  }

  return channel
}

const taxIdentity = sale => {
  return (
    normalizeString(
      sale?.customer_tax_id_number,
    ) ||
    '—'
  )
}

const printStateLabel = sale => {
  return sale?.is_printed
    ? 'Sudah Dicetak'
    : 'Belum Dicetak'
}

const printAuditLabel = sale => {
  if (!sale?.is_printed) {
    return 'Belum pernah dicetak'
  }

  if (sale?.printed_at) {
    return (
      'Cetak terakhir ' +
      formatDateTime(
        sale.printed_at,
      )
    )
  }

  return 'Faktur sudah pernah dicetak'
}

const openInvoice = async sale => {
  if (!sale?.id) {
    return
  }

  invoiceLoading.value = true
  errorMessage.value = ''
  invoiceOpen.value = true
  selectedInvoice.value = null

  try {
    selectedInvoice.value =
      await getFinanceInvoiceByID(
        sale.id,
      )
  } catch (error) {
    invoiceOpen.value = false

    errorMessage.value =
      error.response?.data?.error ||
      'Faktur belum dapat dibuka.'
  } finally {
    invoiceLoading.value = false
  }
}

const closeInvoice = () => {
  invoiceOpen.value = false
  selectedInvoice.value = null
}

const invoiceItems = computed(() => {
  if (
    Array.isArray(
      selectedInvoice.value?.items,
    )
  ) {
    return selectedInvoice.value.items
  }

  if (
    Array.isArray(
      selectedInvoice.value
        ?.invoice_items,
    )
  ) {
    return (
      selectedInvoice.value
        .invoice_items
    )
  }

  return []
})

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

const invoiceIssuer = computed(() => selectedInvoice.value?.issuer_profile || companyProfile)
const invoiceCompanyAddressLines = computed(() => String(invoiceIssuer.value.address || '').split(/\r?\n/).map(line => line.trim()).filter(Boolean))

const invoiceRecipientOverride = computed(() => {
  const orderID =
    selectedInvoice.value
      ?.sales_order_id

  if (!orderID) {
    return null
  }

  const overrides =
    loadRecipientOverrides()

  return (
    overrides?.[
      orderID
    ] ||
    null
  )
})

const invoiceCustomerName = computed(() => {
  return (
    invoiceRecipientOverride
      .value?.name ||
    selectedInvoice.value
      ?.store_name ||
    selectedInvoice.value
      ?.customer_name ||
    '—'
  )
})

const invoiceCustomerAddress = computed(() => {
  return (
    invoiceRecipientOverride
      .value?.address ||
    selectedInvoice.value
      ?.customer_tax_address ||
    selectedInvoice.value
      ?.shipping_address ||
    '—'
  )
})

const invoiceWarehouse = computed(() => {
  return (
    selectedInvoice.value
      ?.warehouse_name ||
    selectedInvoice.value
      ?.warehouse_code ||
    '—'
  )
})

const invoicePrintPages = computed(() => {
  const source =
    invoiceItems.value.map(
      (
        item,
        index,
      ) => ({
        ...item,
        __originalIndex:
          index,
      }),
    )

  const itemsPerPage =
    10

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
    let index = 0;
    index < source.length;
    index += itemsPerPage
  ) {
    pages.push({
      number:
        pages.length + 1,
      items:
        source.slice(
          index,
          index +
            itemsPerPage,
        ),
    })
  }

  return pages.map(
    (
      page,
      index,
    ) => ({
      ...page,
      isFirst:
        index === 0,
      isLast:
        index ===
        pages.length - 1,
    }),
  )
})

const invoicePrintPageCount = computed(() => {
  return (
    invoicePrintPages.value
      .length ||
    1
  )
})

const itemName = item => {
  return (
    item?.product_name ||
    item?.name ||
    '—'
  )
}

const itemCode = item => {
  const productCode =
    item?.product_code ||
    ''

  const sku =
    item?.sku ||
    ''

  if (
    productCode &&
    sku
  ) {
    return `${productCode} · ${sku}`
  }

  return (
    productCode ||
    sku ||
    ''
  )
}

const itemQty = item => {
  return Number(
    item?.quantity ??
      item?.qty ??
      0,
  )
}

const itemUnit = item => {
  return (
    item?.unit ||
    ''
  )
}

const itemPrice = item => {
  return Number(
    item?.unit_price ??
      item?.price ??
      0,
  )
}

const itemDiscountPercent = item => {
  return Number(
    item?.discount_percent ??
      0,
  )
}

const itemDiscountAmount = item => {
  return Number(
    item?.discount_amount ??
      0,
  )
}

const itemTotal = item => {
  const explicit =
    item?.line_total ??
    item?.total ??
    item?.subtotal

  if (
    explicit !== undefined &&
    explicit !== null
  ) {
    return Number(explicit)
  }

  return Math.max(
    itemQty(item) *
      itemPrice(item) -
      itemDiscountAmount(item),
    0,
  )
}

const invoicePaymentChannelLabels = {
  UNSPECIFIED:
    'Belum ditentukan',
  CASH:
    'Tunai',
  BANK_TRANSFER:
    'Transfer Bank',
  VIRTUAL_ACCOUNT:
    'Virtual Account',
  QRIS:
    'QRIS',
  E_WALLET:
    'Dompet Digital',
  CARD:
    'Kartu Debit / Kredit',
  OTHER:
    'Kanal Lainnya',
}

const invoicePaymentSettlementLabel = (
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

const invoicePaymentChannelLabel = value => {
  return (
    invoicePaymentChannelLabels[
      String(
        value ||
        'UNSPECIFIED',
      ).toUpperCase()
    ] ||
    'Belum ditentukan'
  )
}

const invoicePaymentTermsLabel = days => {
  const value =
    Number(
      days ||
      0,
    )

  if (value <= 0) {
    return 'LANGSUNG'
  }

  return `${value} HARI`
}

const terbilangInteger = value => {
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

  const n =
    Math.floor(
      Number(
        value ||
        0,
      ),
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
      Math.floor(
        n / 10,
      ),
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
      Math.floor(
        n / 100,
      ),
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
      Math.floor(
        n / 1000,
      ),
    )} Ribu ${terbilangInteger(
      n % 1000,
    )}`.trim()
  }

  if (n < 1000000000) {
    return `${terbilangInteger(
      Math.floor(
        n / 1000000,
      ),
    )} Juta ${terbilangInteger(
      n % 1000000,
    )}`.trim()
  }

  if (n < 1000000000000) {
    return `${terbilangInteger(
      Math.floor(
        n / 1000000000,
      ),
    )} Miliar ${terbilangInteger(
      n % 1000000000,
    )}`.trim()
  }

  return `${terbilangInteger(
    Math.floor(
      n /
      1000000000000,
    ),
  )} Triliun ${terbilangInteger(
    n %
    1000000000000,
  )}`.trim()
}

const invoiceTerbilang = computed(() => {
  if (!selectedInvoice.value) {
    return ''
  }

  return `${terbilangInteger(
    Math.round(
      Number(
        selectedInvoice.value
          .total ||
        0,
      ),
    ),
  )} Rupiah`
})

const invoiceTaxLabel = computed(() => {
  const invoice =
    selectedInvoice.value

  if (!invoice) {
    return 'PPN'
  }

  if (
    invoice.tax_mode ===
    'STANDARD_PPN'
  ) {
    const nominal =
      Number(
        invoice.tax_rate ??
        invoice.ppn_percent ??
        0,
      )

    const effective =
      Number(
        invoice.effective_tax_rate ??
        0,
      )

    const year =
      invoice.tax_effective_year
        ? ` · ${invoice.tax_effective_year}`
        : ''

    return (
      `PPN ${nominal}%` +
      (
        effective > 0
          ? ` · efektif ${effective}%`
          : ''
      ) +
      year
    )
  }

  return 'PPN'
})

const fetchAllFilteredSales = async () => {
  const first =
    await getFinanceSales({
      ...buildBaseParams(),
      limit:
        EXPORT_BATCH_LIMIT,
      offset:
        0,
    })

  const result =
    Array.isArray(first?.data)
      ? [...first.data]
      : []

  const total =
    Number(
      first?.meta?.total ||
      result.length,
    )

  let offset =
    result.length

  while (
    offset < total
  ) {
    const response =
      await getFinanceSales({
        ...buildBaseParams(),
        limit:
          EXPORT_BATCH_LIMIT,
        offset,
      })

    const rows =
      Array.isArray(
        response?.data,
      )
        ? response.data
        : []

    if (
      rows.length === 0
    ) {
      break
    }

    result.push(
      ...rows,
    )

    offset +=
      rows.length
  }

  return result
}

const buildReportRows = rows => {
  return rows
    .map(
      (
        sale,
        index,
      ) => `
        <tr>
          <td>${index + 1}</td>
          <td>${escapeHTML(formatDate(sale.invoice_date))}</td>
          <td>${escapeHTML(sale.invoice_number || '-')}</td>
          <td>${escapeHTML(sale.order_number || '-')}</td>
          <td>
            <strong>${escapeHTML(sale.store_name || '-')}</strong>
            <br />
            <small>${escapeHTML(sale.customer_code || '')}</small>
          </td>
          <td>${escapeHTML(sale.customer_tax_id_number || '-')}</td>
          <td>${escapeHTML(paymentSettlementLabel(sale.payment_settlement))}</td>
          <td>${escapeHTML(formatDate(sale.due_date))}</td>
          <td class="money">${escapeHTML(formatCurrency(sale.tax_base_amount))}</td>
          <td class="money">${escapeHTML(formatCurrency(sale.tax_amount))}</td>
          <td class="money strong">${escapeHTML(formatCurrency(sale.total))}</td>
        </tr>
      `,
    )
    .join('')
}

const buildPrintHTML = rows => {
  return `
    <!DOCTYPE html>
    <html lang="id">
    <head>
      <meta charset="UTF-8" />
      <title>Rekap Penjualan PT FAS</title>

      <style>
        @page {
          size: A4 landscape;
          margin: 12mm;
        }

        * {
          box-sizing: border-box;
        }

        body {
          margin: 0;
          font-family:
            Arial,
            Helvetica,
            sans-serif;
          color: #111827;
          background: #ffffff;
        }

        .header {
          display: flex;
          align-items: flex-start;
          justify-content: space-between;
          gap: 24px;
          border-bottom: 2px solid #111827;
          padding-bottom: 12px;
        }

        h1 {
          margin: 0;
          font-size: 18px;
        }

        h2 {
          margin: 5px 0 0;
          font-size: 13px;
          font-weight: 600;
        }

        .period {
          font-size: 11px;
          line-height: 1.5;
          text-align: right;
        }

        .summary {
          display: grid;
          grid-template-columns:
            repeat(4, 1fr);
          gap: 8px;
          margin: 14px 0;
        }

        .summary-card {
          border: 1px solid #cbd5e1;
          border-radius: 6px;
          padding: 8px 10px;
        }

        .summary-card span {
          display: block;
          color: #64748b;
          font-size: 9px;
          text-transform: uppercase;
        }

        .summary-card strong {
          display: block;
          margin-top: 4px;
          font-size: 12px;
        }

        table {
          width: 100%;
          border-collapse: collapse;
          table-layout: auto;
        }

        th,
        td {
          border: 1px solid #9ca3af;
          padding: 5px 6px;
          font-size: 8.5px;
          vertical-align: top;
        }

        th {
          background: #f1f5f9;
          font-weight: 700;
          text-align: left;
        }

        td.money {
          white-space: nowrap;
          text-align: right;
        }

        td.strong {
          font-weight: 700;
        }

        small {
          color: #64748b;
          font-size: 7px;
        }

        .footer {
          margin-top: 12px;
          color: #64748b;
          font-size: 8px;
        }
      </style>
    </head>

    <body>${companyDocumentHTML()}
      <div class="header">
        <div>
          <h1>${escapeHTML(companyProfile.name)}</h1>
          <h2>Rekap Penjualan</h2>
        </div>

        <div class="period">
          <strong>Periode</strong><br />
          ${escapeHTML(periodLabel.value)}
          <br />
          Dicetak:
          ${escapeHTML(formatDateTime(new Date()))}
        </div>
      </div>

      <section class="summary">
        <div class="summary-card">
          <span>Total Penjualan</span>
          <strong>${escapeHTML(formatCurrency(summary.value.grand_total))}</strong>
        </div>

        <div class="summary-card">
          <span>Jumlah Faktur</span>
          <strong>${escapeHTML(formatNumber(summary.value.invoice_count))}</strong>
        </div>

        <div class="summary-card">
          <span>DPP</span>
          <strong>${escapeHTML(formatCurrency(summary.value.tax_base_amount))}</strong>
        </div>

        <div class="summary-card">
          <span>PPN</span>
          <strong>${escapeHTML(formatCurrency(summary.value.tax_amount))}</strong>
        </div>
      </section>

      <table>
        <thead>
          <tr>
            <th>No</th>
            <th>Tanggal</th>
            <th>No. Faktur</th>
            <th>No. SO</th>
            <th>Customer</th>
            <th>NPWP / NIK</th>
            <th>Skema</th>
            <th>Jatuh Tempo</th>
            <th>DPP</th>
            <th>PPN</th>
            <th>Total</th>
          </tr>
        </thead>

        <tbody>
          ${buildReportRows(rows)}
        </tbody>
      </table>

      <div class="footer">
        Data mengikuti filter Finance yang aktif pada saat laporan dibuat.
      </div>

      <script>
        window.onload = () => {
          window.print()
        }
      <\/script>
    </body>
    </html>
  `
}

const openSalesReportPreview = async () => {
  reportPreviewOpen.value = true
  reportPreviewLoading.value = true
  reportPreviewRows.value = []
  errorMessage.value = ''

  try {
    reportPreviewRows.value =
      await fetchAllFilteredSales()
  } catch (error) {
    reportPreviewOpen.value = false

    errorMessage.value =
      error.response?.data?.error ||
      error.message ||
      'Preview laporan belum dapat dimuat.'
  } finally {
    reportPreviewLoading.value = false
  }
}

const closeSalesReportPreview = () => {
  if (reportPreviewLoading.value) {
    return
  }

  reportPreviewOpen.value = false
  reportPreviewRows.value = []
}

const printSalesReport = async () => {
  exporting.value = true
  errorMessage.value = ''

  try {
    const rows =
      reportPreviewRows.value.length > 0
        ? reportPreviewRows.value
        : await fetchAllFilteredSales()

    const win =
      window.open(
        '',
        '_blank',
      )

    if (!win) {
      throw new Error(
        'Pop-up browser diblokir. Izinkan pop-up untuk mencetak laporan.',
      )
    }

    win.opener = null
    win.document.open()
    win.document.write(
      buildPrintHTML(rows),
    )
    win.document.close()
  } catch (error) {
    errorMessage.value =
      error.response?.data?.error ||
      error.message ||
      'Laporan belum dapat dicetak.'
  } finally {
    exporting.value = false
  }
}

const excelCell = value => {
  return escapeHTML(
    String(
      value ?? '',
    ),
  )
}

const exportExcel = async () => {
  exporting.value = true
  errorMessage.value = ''

  try {
    const rows =
      await fetchAllFilteredSales()

    const body =
      rows
        .map(
          (
            sale,
            index,
          ) => `
            <tr>
              <td>${index + 1}</td>
              <td>${excelCell(formatDate(sale.invoice_date))}</td>
              <td>${excelCell(sale.invoice_number)}</td>
              <td>${excelCell(sale.delivery_note_number)}</td>
              <td>${excelCell(sale.order_number)}</td>
              <td>${excelCell(sale.customer_code)}</td>
              <td>${excelCell(sale.store_name)}</td>
              <td>${excelCell(sale.customer_phone)}</td>
              <td>${excelCell(sale.customer_tax_id_number)}</td>
              <td>${excelCell(sale.warehouse_name)}</td>
              <td>${excelCell(paymentSettlementLabel(sale.payment_settlement))}</td>
              <td>${excelCell(paymentChannelLabel(sale))}</td>
              <td>${excelCell(formatDate(sale.due_date))}</td>
              <td>${Number(sale.total_qty || 0)}</td>
              <td>${Number(sale.subtotal || 0)}</td>
              <td>${Number(sale.discount_amount || 0)}</td>
              <td>${Number(sale.net_sales || 0)}</td>
              <td>${Number(sale.tax_base_amount || 0)}</td>
              <td>${Number(sale.tax_amount || 0)}</td>
              <td>${Number(sale.total || 0)}</td>
              <td>${excelCell(printStateLabel(sale))}</td>
            </tr>
          `,
        )
        .join('')

    const workbook = `
      <html xmlns:o="urn:schemas-microsoft-com:office:office"
        xmlns:x="urn:schemas-microsoft-com:office:excel"
        xmlns="http://www.w3.org/TR/REC-html40">
      <head>
        <meta charset="UTF-8" />
        <style>
          table {
            border-collapse: collapse;
          }

          th,
          td {
            border: 1px solid #999;
            padding: 5px;
          }

          th {
            font-weight: bold;
            background: #e2e8f0;
          }
        </style>
      </head>

      <body>${companyDocumentHTML()}
        <table>
          <tr>
            <th colspan="21">
              ${escapeHTML(companyProfile.name)} - REKAP PENJUALAN
            </th>
          </tr>

          <tr>
            <td colspan="21">
              Periode ${excelCell(periodLabel.value)}
            </td>
          </tr>

          <tr>
            <td colspan="21">
              Total Penjualan ${excelCell(formatCurrency(summary.value.grand_total))}
              | Faktur ${excelCell(formatNumber(summary.value.invoice_count))}
              | DPP ${excelCell(formatCurrency(summary.value.tax_base_amount))}
              | PPN ${excelCell(formatCurrency(summary.value.tax_amount))}
            </td>
          </tr>

          <tr>
            <th>No</th>
            <th>Tanggal</th>
            <th>No Faktur</th>
            <th>No SJ</th>
            <th>No SO</th>
            <th>Kode Customer</th>
            <th>Customer</th>
            <th>Telepon</th>
            <th>NPWP / NIK</th>
            <th>Gudang</th>
            <th>Skema Bayar</th>
            <th>Kanal Bayar</th>
            <th>Jatuh Tempo</th>
            <th>Total Qty</th>
            <th>Subtotal</th>
            <th>Diskon</th>
            <th>Penjualan Bersih</th>
            <th>DPP</th>
            <th>PPN</th>
            <th>Total</th>
            <th>Status Cetak</th>
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
      `rekap-penjualan-${filters.date_from}-${filters.date_to}.xls`

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
      error.response?.data?.error ||
      error.message ||
      'Data belum dapat diekspor ke Excel.'
  } finally {
    exporting.value = false
  }
}

const buildInvoicePageRowsHTML = page => {
  if (
    !Array.isArray(
      page?.items,
    ) ||
    page.items.length === 0
  ) {
    return `
      <tr>
        <td
          colspan="9"
          class="empty-items"
        >
          Detail item belum tersedia.
        </td>
      </tr>
    `
  }

  return page.items
    .map(
      item => `
        <tr>
          <td>
            ${
              Number(
                item.__originalIndex ??
                0,
              ) +
              1
            }
          </td>

          <td class="product">
            <strong>
              ${escapeHTML(
                itemName(item),
              )}
            </strong>

            ${
              itemCode(item)
                ? `
                  <small>
                    ${escapeHTML(
                      itemCode(item),
                    )}
                  </small>
                `
                : ''
            }
          </td>

          <td>
            ${escapeHTML(
              item?.batch_number ||
              '-',
            )}
          </td>

          <td>
            ${escapeHTML(
              formatDate(
                item?.expiry_date,
              ),
            )}
          </td>

          <td>
            ${escapeHTML(
              `${itemQty(item)} ${itemUnit(item)}`.trim(),
            )}
          </td>

          <td>
            ${escapeHTML(
              formatCurrency(
                itemPrice(item),
              ),
            )}
          </td>

          <td>
            ${escapeHTML(
              itemDiscountPercent(
                item,
              ),
            )}
          </td>

          <td>
            ${escapeHTML(
              formatCurrency(
                itemDiscountAmount(
                  item,
                ),
              ),
            )}
          </td>

          <td>
            ${escapeHTML(
              formatCurrency(
                itemTotal(item),
              ),
            )}
          </td>
        </tr>
      `,
    )
    .join('')
}

const buildInvoicePageHTML = (
  invoice,
  page,
  totalPages,
) => {
  const companyAddress =
    invoiceCompanyAddressLines
      .value
      .map(
        line => `
          <span>
            ${escapeHTML(line)}
          </span>
        `,
      )
      .join('')

  const provider =
    invoice?.payment_provider
      ? `
        <span>
          ${escapeHTML(
            invoice.payment_provider,
          )}
        </span>
      `
      : ''

  const warehouseClass =
    invoice
      ?.warehouse_classification
      ? `
        <span>
          ${escapeHTML(
            invoice
              .warehouse_classification,
          )}
        </span>
      `
      : ''

  const taxIdentity =
    invoice
      ?.customer_tax_id_number
      ? `
        <span>
          NPWP/NIK :
          ${escapeHTML(
            invoice
              .customer_tax_id_number,
          )}
        </span>
      `
      : ''

  const continuation =
    !page.isLast
      ? `
        <div class="continuation">
          <span>
            Bersambung ke halaman
            ${page.number + 1}
          </span>

          <strong>
            ${escapeHTML(
              invoice?.invoice_number ||
              '-',
            )}
          </strong>
        </div>
      `
      : ''

  const lastPage =
    page.isLast
      ? `
        <section class="bottom-grid">
          <div class="signature-box">
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

          <div class="tax-box">
            <div>
              <span>Total</span>

              <strong>
                ${escapeHTML(
                  formatCurrency(
                    invoice?.subtotal,
                  ),
                )}
              </strong>
            </div>

            <div>
              <span>Discount</span>

              <strong>
                ${escapeHTML(
                  formatCurrency(
                    invoice
                      ?.discount_amount,
                  ),
                )}
              </strong>
            </div>

            <div>
              <span>
                DPP
                ${
                  invoice?.tax_mode ===
                  'STANDARD_PPN'
                    ? 'Nilai Lain'
                    : ''
                }
              </span>

              <strong>
                ${escapeHTML(
                  formatCurrency(
                    invoice
                      ?.tax_base_amount,
                  ),
                )}
              </strong>
            </div>

            <div>
              <span>
                ${escapeHTML(
                  invoiceTaxLabel.value,
                )}
              </span>

              <strong>
                ${escapeHTML(
                  formatCurrency(
                    invoice?.ppn_amount ??
                    invoice?.tax_amount,
                  ),
                )}
              </strong>
            </div>

            <div class="tax-total">
              <span>Jumlah</span>

              <strong>
                ${escapeHTML(
                  formatCurrency(
                    invoice?.total,
                  ),
                )}
              </strong>
            </div>
          </div>
        </section>

        <section class="paper-footer">
          <div class="terbilang">
            <strong>
              Nilai dalam Huruf :
            </strong>

            <span>
              ${escapeHTML(
                invoiceTerbilang.value,
              )}
            </span>
          </div>
        </section>


      `
      : ''

  return `
    <article class="paper">
      <header class="original-header">
        <div class="company-box">
          <strong>
            ${escapeHTML(invoiceIssuer.value.name)}
          </strong>

          ${companyAddress}

          <span>
            No. Izin :
            ${escapeHTML(invoiceIssuer.value.license_number)}
          </span>

          <span>
            NPWP :
            ${escapeHTML(invoiceIssuer.value.tax_id)}
          </span>
        </div>

        <div class="middle-meta">
          <div>
            <span>No. SJ</span>

            <strong>
              ${escapeHTML(
                invoice
                  ?.delivery_note_number ||
                '-',
              )}
            </strong>
          </div>

          <div>
            <span>Tgl.</span>

            <strong>
              ${escapeHTML(
                formatDate(
                  invoice?.invoice_date,
                ),
              )}
            </strong>
          </div>

          <div>
            <span>
              Tgl J. Tempo
            </span>

            <strong>
              ${escapeHTML(
                formatDate(
                  invoice?.due_date,
                ),
              )}
            </strong>
          </div>

          <div>
            <span>Salesman</span>

            <strong>
              ${escapeHTML(
                invoice?.sales_name ||
                '-',
              )}
            </strong>
          </div>

          <div>
            <span>
              Skema Bayar
            </span>

            <strong>
              ${escapeHTML(
                invoicePaymentSettlementLabel(
                  invoice
                    ?.payment_settlement,
                  invoice
                    ?.payment_channel,
                ),
              )}
            </strong>
          </div>

          <div>
            <span>
              Kanal Bayar
            </span>

            <strong>
              ${escapeHTML(
                invoicePaymentChannelLabel(
                  invoice
                    ?.payment_channel,
                ),
              )}
            </strong>

            ${provider}
          </div>

          <div>
            <span>Tempo</span>

            <strong>
              ${escapeHTML(
                invoice
                  ?.payment_settlement ===
                'TERM'
                  ? invoicePaymentTermsLabel(
                      invoice
                        ?.payment_terms_days,
                    )
                  : 'LANGSUNG',
              )}
            </strong>
          </div>

          <div>
            <span>Gudang</span>

            <strong>
              ${escapeHTML(
                invoiceWarehouse.value,
              )}
            </strong>

            ${warehouseClass}
          </div>
        </div>

        <div class="customer-box">
          <span>
            Kepada Yth,
          </span>

          <strong>
            ${escapeHTML(
              invoiceCustomerName.value,
            )}
          </strong>

          <span class="customer-address">
            ${escapeHTML(
              invoiceCustomerAddress.value,
            )}
          </span>

          ${taxIdentity}

          <span>
            ${escapeHTML(
              invoice
                ?.customer_code ||
              '',
            )}
          </span>
        </div>
      </header>

      <div class="reference-row">
        <div>
          <span>
            No. Faktur
          </span>

          <strong>
            ${escapeHTML(
              invoice
                ?.invoice_number ||
              '-',
            )}
          </strong>
        </div>

        <div>
          <span>
            No. Sales Order
          </span>

          <strong>
            ${escapeHTML(
              invoice
                ?.order_number ||
              '-',
            )}
          </strong>
        </div>

        <div>
          <span>Halaman</span>

          <strong>
            ${page.number}
            /
            ${totalPages}
          </strong>
        </div>
      </div>

      <table class="paper-table">
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
          ${buildInvoicePageRowsHTML(page)}
        </tbody>
      </table>

      ${continuation}
      ${lastPage}
    </article>
  `
}

const buildInvoicePrintHTML = invoice => {
  const pages =
    invoicePrintPages.value

  const pageHTML =
    pages
      .map(
        page =>
          buildInvoicePageHTML(
            invoice,
            page,
            pages.length,
          ),
      )
      .join('')

  return `
    <!DOCTYPE html>
    <html lang="id">
    <head>
      <meta charset="UTF-8" />

      <title>
        ${escapeHTML(
          invoice?.invoice_number ||
          'Faktur',
        )}
      </title>

      <style>
        @page {
          size: 9.5in 5.5in;
          margin: 0;
        }

        * {
          box-sizing: border-box;
        }

        html,
        body {
          margin: 0;
          padding: 0;
          background: #ffffff;
        }

        body {
          font-family:
            "Courier New",
            Courier,
            monospace;
          color: #1f2937;
        }

        .paper {
          width: 9.5in;
          height: 5.5in;
          overflow: hidden;
          background: #fffef6;
          padding: 0.10in 0.12in;
          page-break-after: always;
          break-after: page;
        }

        .paper:last-child {
          page-break-after: auto;
          break-after: auto;
        }

        .original-header {
          display: grid;
          grid-template-columns:
            1.15fr
            0.9fr
            1.15fr;
          gap: 0.04in;
        }

        .company-box,
        .customer-box {
          display: flex;
          min-height: 0.78in;
          flex-direction: column;
          gap: 0.01in;
          border: 1px solid #708090;
          padding: 0.035in 0.05in;
          font-size: 5.5pt;
          line-height: 1.15;
        }

        .company-box strong,
        .customer-box strong {
          font-size: 6.3pt;
        }

        .customer-address {
          white-space: pre-line;
        }

        .middle-meta {
          display: grid;
          grid-template-columns:
            0.75fr
            1.25fr;
          align-content: start;
          gap:
            0.012in
            0.04in;
          font-size: 5.2pt;
          line-height: 1.08;
        }

        .middle-meta > div {
          display: contents;
        }

        .middle-meta span {
          color: #64748b;
        }

        .middle-meta strong {
          color: #111827;
          font-size: 5.25pt;
        }

        .reference-row {
          display: grid;
          grid-template-columns:
            1fr
            1fr
            1fr;
          margin-top: 0.035in;
        }

        .reference-row > div {
          min-height: 0.32in;
          border:
            1px solid
            #708090;
          padding:
            0.025in
            0.045in;
          font-size: 5.1pt;
        }

        .reference-row > div + div {
          border-left: 0;
        }

        .reference-row span {
          display: block;
          color: #64748b;
          font-size: 4.7pt;
        }

        .reference-row strong {
          display: block;
          margin-top: 0.012in;
          color: #111827;
          font-size: 5.35pt;
        }

        .paper-table {
          width: 100%;
          border-collapse: collapse;
          table-layout: fixed;
        }

        .paper-table th,
        .paper-table td {
          border: 1px solid #708090;
          padding:
            0.018in
            0.025in;
          font-size: 4.7pt;
          line-height: 1.05;
          vertical-align: top;
        }

        .paper-table th {
          background: #eef2f7;
          font-size: 4.35pt;
          text-align: center;
          text-transform: uppercase;
        }

        .paper-table th:nth-child(1),
        .paper-table td:nth-child(1) {
          width: 6%;
          text-align: center;
        }

        .paper-table th:nth-child(2),
        .paper-table td:nth-child(2) {
          width: 27%;
        }

        .paper-table th:nth-child(3),
        .paper-table td:nth-child(3) {
          width: 10%;
        }

        .paper-table th:nth-child(4),
        .paper-table td:nth-child(4) {
          width: 8%;
        }

        .paper-table th:nth-child(5),
        .paper-table td:nth-child(5) {
          width: 11%;
        }

        .paper-table th:nth-child(6),
        .paper-table td:nth-child(6) {
          width: 12%;
        }

        .paper-table th:nth-child(7),
        .paper-table td:nth-child(7) {
          width: 8%;
        }

        .paper-table th:nth-child(8),
        .paper-table td:nth-child(8) {
          width: 9%;
        }

        .paper-table th:nth-child(9),
        .paper-table td:nth-child(9) {
          width: 12%;
        }

        .product strong,
        .product small {
          display: block;
        }

        .product small {
          margin-top: 0.012in;
          color: #64748b;
          font-size: 3.8pt;
        }

        .empty-items {
          padding: 0.18in !important;
          color: #64748b;
          text-align: center;
        }

        .continuation {
          display: flex;
          justify-content: space-between;
          border:
            1px solid
            #708090;
          border-top: 0;
          padding:
            0.035in
            0.05in;
          font-size: 4.7pt;
        }

        .bottom-grid {
          display: grid;
          grid-template-columns:
            minmax(0, 1fr)
            2.3in;
          margin-top: 0.035in;
        }

        .signature-box {
          display: grid;
          min-height: 0.62in;
          grid-template-columns:
            1fr
            1fr;
          border: 1px solid #708090;
        }

        .signature-box > div {
          display: flex;
          flex-direction: column;
          justify-content: space-between;
          padding:
            0.04in
            0.05in;
          font-size: 4.7pt;
          text-align: center;
        }

        .signature-box > div:first-child {
          border-right:
            1px solid
            #708090;
        }

        .signature-box span {
          color: #64748b;
        }

        .signature-box strong {
          font-size: 4.5pt;
        }

        .tax-box {
          border:
            1px solid
            #708090;
          border-left: 0;
        }

        .tax-box > div {
          display: flex;
          min-height: 0.12in;
          align-items: center;
          justify-content: space-between;
          gap: 0.08in;
          border-bottom:
            1px solid
            #94a3b8;
          padding:
            0.018in
            0.04in;
          font-size: 4.5pt;
        }

        .tax-box > div:last-child {
          border-bottom: 0;
        }

        .tax-box strong {
          white-space: nowrap;
        }

        .tax-total {
          font-weight: 800;
        }

        .paper-footer {
          display: block;
          min-height: 0.42in;
          border:
            1px solid
            #708090;
          border-top: 0;
        }

        .terbilang {
          display: flex;
          gap: 0.08in;
          padding:
            0.055in
            0.06in;
          font-size: 4.35pt;
          line-height: 1.15;
        }

        .terbilang strong {
          white-space: nowrap;
        }

        .paper-note {
          border:
            1px solid
            #708090;
          border-top: 0;
          padding:
            0.025in
            0.05in;
          font-size: 4.1pt;
        }
      </style>
    </head>

    <body>
      ${pageHTML}

      <script>
        window.onload = () => {
          window.print()
        }
      <\/script>
    </body>
    </html>
  `
}

const printSelectedInvoice = async () => {
  if (
    !selectedInvoice.value?.id ||
    invoicePrinting.value
  ) {
    return
  }

  /*
    Preview / Lihat Faktur tidak menambah hitungan cetak.
    Hitungan baru dicatat ketika user menekan tombol Cetak / PDF.

    Browser tidak memberi konfirmasi yang andal apakah dialog print
    akhirnya benar-benar dicetak, disimpan sebagai PDF, atau dibatalkan.
    Karena itu definisi audit yang dipakai adalah "permintaan cetak":
    satu klik Cetak / PDF = satu aktivitas cetak.
  */

  const win =
    window.open(
      '',
      '_blank',
    )

  if (!win) {
    errorMessage.value =
      'Pop-up browser diblokir. Izinkan pop-up untuk mencetak faktur.'
    return
  }

  invoicePrinting.value = true
  errorMessage.value = ''

  win.document.open()
  win.document.write(`
    <!DOCTYPE html>
    <html>
      <head>
        <meta charset="UTF-8" />
        <title>Menyiapkan faktur...</title>
      </head>
      <body
        style="
          margin:0;
          min-height:100vh;
          display:grid;
          place-items:center;
          font-family:Arial,sans-serif;
          color:#334155;
        "
      >
        Menyiapkan faktur...
      </body>
    </html>
  `)
  win.document.close()

  try {
    const updated =
      await markInvoicePrinted(
        selectedInvoice.value.id,
      )

    selectedInvoice.value = {
      ...selectedInvoice.value,
      ...updated,
    }

    const saleIndex =
      sales.value.findIndex(
        item =>
          item.id ===
          selectedInvoice.value.id,
      )

    if (saleIndex >= 0) {
      sales.value[
        saleIndex
      ] = {
        ...sales.value[
          saleIndex
        ],
        is_printed: true,
        printed_at:
          updated?.printed_at ||
          new Date()
            .toISOString(),
        print_count:
          updated?.print_count ??
          sales.value[
            saleIndex
          ]?.print_count ??
          1,
        print_version:
          updated?.print_version ??
          sales.value[
            saleIndex
          ]?.print_version ??
          1,
      }
    }

    win.opener = null
    win.document.open()
    win.document.write(
      buildInvoicePrintHTML(
        selectedInvoice.value,
      ),
    )
    win.document.close()

    await loadFinanceSales({
      silent: true,
    })
  } catch (error) {
    try {
      win.close()
    } catch {
      // Browser may already have closed the print window.
    }

    errorMessage.value =
      error.response?.data?.error ||
      'Faktur belum dapat dicetak. Silakan coba kembali.'
  } finally {
    invoicePrinting.value = false
  }
}

watch(
  () => filters.query,
  () => {
    if (searchTimer) {
      clearTimeout(
        searchTimer,
      )
    }

    searchTimer =
      setTimeout(
        async () => {
          resetPagination()

          await loadFinanceSales()
        },
        450,
      )
  },
)

watch(
  () => [
    filters.date_from,
    filters.date_to,
  ],
  () => {
    activePreset.value =
      'CUSTOM'
  },
)

useModalDialog(
  () => customerModalOpen.value
    ? '.customer-modal'
    : reportPreviewOpen.value
      ? '.report-preview-modal'
      : invoiceOpen.value
        ? '.invoice-modal'
        : null,
  () => {
    if (customerModalOpen.value) {
      closeCustomerEditor()
    } else if (reportPreviewOpen.value) {
      closeSalesReportPreview()
    } else if (invoiceOpen.value) {
      closeInvoice()
    }
  },
)

onMounted(async () => {
  await loadCompanyProfile().catch(failure => { console.error('Profil perusahaan belum termuat') })
  await Promise.all([
    loadFinanceSales(),
    loadCustomers(),
  ])

  refreshTimer =
    window.setInterval(
      () => {
        loadFinanceSales({
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

  if (searchTimer) {
    window.clearTimeout(
      searchTimer,
    )
  }
})
</script>

<template>
  <div class="finance-page finance-customers-page">
    <header class="finance-header">
      <div>
        <div class="finance-kicker-row">
          <p class="finance-kicker">
            Finance & Accounting
          </p>

          <span class="live-badge">
            <span class="live-dot"></span>
            LIVE
          </span>
        </div>

        <h1 class="finance-title">
          Customer Master
        </h1>

        <p class="finance-description">
          Kelola toko, alamat, telepon, NPWP/NIK, PKP, TKU
          dan tempo default dalam satu Customer Master.
        </p>

        <p class="last-update">
          {{ realTimeLabel }}
        </p>
      </div>

      <div class="header-actions">
        <button
          type="button"
          class="secondary-button"
          :disabled="
            refreshing ||
            loading
          "
          @click="refreshNow"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path
              d="M20 11a8 8 0 1 0 2 5"
            />
            <path
              d="M20 4v7h-7"
            />
          </svg>

          {{
            refreshing
              ? 'Memperbarui...'
              : 'Refresh'
          }}
        </button>

        <button
          type="button"
          class="secondary-button"
          :disabled="exporting"
          @click="openSalesReportPreview"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path
              d="M6 9V2h12v7"
            />
            <path
              d="M6 18H4a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h16a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"
            />
            <path
              d="M6 14h12v8H6z"
            />
          </svg>

          Print / PDF
        </button>

        <button
          type="button"
          class="primary-button"
          :disabled="exporting"
          @click="exportExcel"
        >
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path
              d="M12 3v12"
            />
            <path
              d="m7 10 5 5 5-5"
            />
            <path
              d="M5 21h14"
            />
          </svg>

          {{
            exporting
              ? 'Menyiapkan...'
              : 'Export Excel'
          }}
        </button>
      </div>
    </header>

    <FinanceNavigation />

    <div
      v-if="errorMessage"
      class="message message-error"
    >
      {{ errorMessage }}
    </div>

    <div
      v-if="successMessage"
      class="message message-success"
    >
      {{ successMessage }}
    </div>

    <section class="summary-grid">
      <article class="summary-card summary-main">
        <div class="summary-icon">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path
              d="M4 19V5"
            />
            <path
              d="M4 19h16"
            />
            <path
              d="m7 15 4-4 3 3 5-6"
            />
          </svg>
        </div>

        <p class="summary-label">
          Total Penjualan
        </p>

        <p class="summary-value">
          {{
            formatCompactCurrency(
              summary.grand_total,
            )
          }}
        </p>

        <p class="summary-detail">
          {{
            formatCurrency(
              summary.grand_total,
            )
          }}
        </p>
      </article>

      <article class="summary-card">
        <div class="summary-icon">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path
              d="M6 3h12v18H6z"
            />
            <path
              d="M9 7h6M9 11h6M9 15h4"
            />
          </svg>
        </div>

        <p class="summary-label">
          Jumlah Faktur
        </p>

        <p class="summary-value">
          {{
            formatNumber(
              summary.invoice_count,
            )
          }}
        </p>

        <p class="summary-detail">
          {{
            formatNumber(
              summary.total_qty,
            )
          }}
          total qty
        </p>
      </article>

      <article class="summary-card">
        <div class="summary-icon">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <circle
              cx="9"
              cy="8"
              r="4"
            />
            <path
              d="M2 21a7 7 0 0 1 14 0"
            />
            <path
              d="M17 11a4 4 0 0 1 5 4"
            />
          </svg>
        </div>

        <p class="summary-label">
          Customer
        </p>

        <p class="summary-value">
          {{
            formatNumber(
              summary.customer_count,
            )
          }}
        </p>

        <p class="summary-detail">
          customer pada periode aktif
        </p>
      </article>

      <article class="summary-card">
        <div class="summary-icon">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <path
              d="M5 5h14v14H5z"
            />
            <path
              d="M8 9h8M8 13h4"
            />
          </svg>
        </div>

        <p class="summary-label">
          PPN
        </p>

        <p class="summary-value">
          {{
            formatCompactCurrency(
              summary.tax_amount,
            )
          }}
        </p>

        <p class="summary-detail">
          DPP
          {{
            formatCurrency(
              summary.tax_base_amount,
            )
          }}
        </p>
      </article>
    </section>

    <section class="analytics-panel">
      <div class="section-heading">
        <div>
          <p class="section-kicker">
            Real-time Analytics
          </p>

          <h2>
            Statistik Penjualan
          </h2>
        </div>

        <span class="period-chip">
          {{ periodLabel }}
        </span>
      </div>

      <div class="analytics-grid">
        <article class="metric-box">
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
        </article>

        <article class="metric-box">
          <span>
            Rata-rata Faktur
          </span>

          <strong>
            {{
              formatCurrency(
                summary.average_invoice,
              )
            }}
          </strong>

          <small>
            rata-rata nilai per faktur
          </small>
        </article>

        <article class="metric-box">
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
        </article>

        <article class="metric-box">
          <span>
            Subtotal
          </span>

          <strong>
            {{
              formatCurrency(
                summary.subtotal,
              )
            }}
          </strong>

          <small>
            sebelum diskon dan pajak
          </small>
        </article>
      </div>

      <div class="ratio-grid">
        <article class="ratio-card">
          <div class="ratio-heading">
            <div>
              <span>Penjualan Tempo</span>
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
              {{ termRatio }}%
            </b>
          </div>

          <div class="ratio-track">
            <span
              :style="{
                width:
                  `${termRatio}%`,
              }"
            ></span>
          </div>
        </article>

        <article class="ratio-card">
          <div class="ratio-heading">
            <div>
              <span>Pembayaran Langsung</span>
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
              {{ immediateRatio }}%
            </b>
          </div>

          <div class="ratio-track">
            <span
              :style="{
                width:
                  `${immediateRatio}%`,
              }"
            ></span>
          </div>
        </article>

        <article class="ratio-card">
          <div class="ratio-heading">
            <div>
              <span>Faktur Tercetak</span>
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
              {{ printedRatio }}%
            </b>
          </div>

          <div class="ratio-track">
            <span
              :style="{
                width:
                  `${printedRatio}%`,
              }"
            ></span>
          </div>
        </article>
      </div>
    </section>

    <section class="filters-panel">
      <div class="filter-top-row">
        <div class="search-shell">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <circle
              cx="11"
              cy="11"
              r="7"
            />
            <path
              d="m20 20-3.5-3.5"
            />
          </svg>

          <input
            v-model="filters.query"
            type="search"
            placeholder="Cari toko, faktur, SO, NPWP, telepon, gudang..."
          />

          <button
            v-if="filters.query"
            type="button"
            class="search-clear"
            aria-label="Hapus pencarian"
            @click="clearSearch"
          >
            <FasIcon name="close" />
          </button>
        </div>

        <FasSelect
          v-model="filters.print_state"
          class="finance-select"
          @change="handlePrintStateChange"
        >
          <option value="all">
            Semua Faktur
          </option>

          <option value="printed">
            Sudah Dicetak
          </option>

          <option value="unprinted">
            Belum Dicetak
          </option>
        </FasSelect>
      </div>

      <div class="preset-row">
        <button
          type="button"
          :class="[
            'preset-button',
            activePreset === 'TODAY'
              ? 'preset-active'
              : '',
          ]"
          @click="
            applyPreset(
              'TODAY',
            )
          "
        >
          Hari Ini
        </button>

        <button
          type="button"
          :class="[
            'preset-button',
            activePreset ===
            'THIS_MONTH'
              ? 'preset-active'
              : '',
          ]"
          @click="
            applyPreset(
              'THIS_MONTH',
            )
          "
        >
          Bulan Ini
        </button>

        <button
          type="button"
          :class="[
            'preset-button',
            activePreset ===
            'LAST_30_DAYS'
              ? 'preset-active'
              : '',
          ]"
          @click="
            applyPreset(
              'LAST_30_DAYS',
            )
          "
        >
          30 Hari
        </button>

        <button
          type="button"
          :class="[
            'preset-button',
            activePreset ===
            'SIX_MONTHS'
              ? 'preset-active'
              : '',
          ]"
          @click="
            applyPreset(
              'SIX_MONTHS',
            )
          "
        >
          6 Bulan
        </button>

        <button
          type="button"
          :class="[
            'preset-button',
            activePreset ===
            'THIS_YEAR'
              ? 'preset-active'
              : '',
          ]"
          @click="
            applyPreset(
              'THIS_YEAR',
            )
          "
        >
          Tahun Ini
        </button>

        <div class="custom-period">
          <label>
            <span>Dari</span>

            <input
              v-model="filters.date_from"
              type="date"
            />
          </label>

          <span class="period-separator">
            →
          </span>

          <label>
            <span>Sampai</span>

            <input
              v-model="filters.date_to"
              type="date"
            />
          </label>

          <button
            type="button"
            class="apply-period-button"
            @click="useCustomPeriod"
          >
            Terapkan
          </button>
        </div>
      </div>
    </section>

    <section class="sales-panel">
      <div class="section-heading sales-heading">
        <div>
          <p class="section-kicker">
            Sales Archive
          </p>

          <h2>
            Data Penjualan
          </h2>

          <p class="table-description">
            {{
              formatNumber(
                meta.total,
              )
            }}
            faktur ditemukan.
          </p>
        </div>

        <div class="sales-heading-actions">
          <div class="table-status">
            <span>
              Halaman
              {{ currentPage }}
              /
              {{ totalPages }}
            </span>
          </div>

          <button
            type="button"
            class="table-report-button"
            :disabled="exporting"
            @click="openSalesReportPreview"
          >
            Preview / Print
          </button>

          <button
            type="button"
            class="table-excel-button"
            :disabled="exporting"
            @click="exportExcel"
          >
            Excel
          </button>
        </div>
      </div>

      <div
        v-if="loading"
        class="loading-state"
      >
        <span class="loading-spinner"></span>

        <p>
          Memuat data penjualan...
        </p>
      </div>

      <div
        v-else-if="
          sales.length === 0
        "
        class="empty-state"
      >
        <div class="empty-icon">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.7"
          >
            <path
              d="M6 3h12v18H6z"
            />
            <path
              d="M9 8h6M9 12h6M9 16h4"
            />
          </svg>
        </div>

        <h3>
          Belum ada data pada periode ini
        </h3>

        <p>
          Ubah filter tanggal atau pencarian untuk melihat arsip penjualan lainnya.
        </p>
      </div>

      <div
        v-else
        class="table-scroll-shell"
      >
        <table class="finance-table">
          <thead>
            <tr>
              <th>Tanggal</th>
              <th>Faktur</th>
              <th>Customer</th>
              <th>NPWP / NIK</th>
              <th>Sales Order</th>
              <th>Pembayaran</th>
              <th>Jatuh Tempo</th>
              <th>DPP</th>
              <th>PPN</th>
              <th>Total</th>
              <th>Status Cetak</th>
              <th>Aksi</th>
            </tr>
          </thead>

          <tbody>
            <tr
              v-for="
                sale in sales
              "
              :key="sale.id"
            >
              <td>
                <strong class="date-value">
                  {{
                    formatDate(
                      sale.invoice_date,
                    )
                  }}
                </strong>

                <small>
                  {{
                    sale.delivery_note_number ||
                    'Tanpa No. SJ'
                  }}
                </small>
              </td>

              <td>
                <strong class="invoice-number">
                  {{
                    sale.invoice_number
                  }}
                </strong>

                <small>
                  {{
                    sale.is_printed
                      ? `Versi ${sale.print_version || 1}`
                      : 'Draft arsip'
                  }}
                </small>
              </td>

              <td>
                <strong>
                  {{
                    sale.store_name
                  }}
                </strong>

                <small>
                  {{
                    sale.customer_code ||
                    '—'
                  }}
                </small>

                <small
                  v-if="
                    sale.customer_phone
                  "
                >
                  {{
                    sale.customer_phone
                  }}
                </small>
              </td>

              <td>
                <span class="tax-id">
                  {{
                    taxIdentity(
                      sale,
                    )
                  }}
                </span>
              </td>

              <td>
                <strong>
                  {{
                    sale.order_number ||
                    '—'
                  }}
                </strong>

                <small>
                  {{
                    sale.warehouse_name ||
                    sale.warehouse_code ||
                    '—'
                  }}
                </small>
              </td>

              <td>
                <span
                  :class="[
                    'settlement-badge',
                    String(
                      sale.payment_settlement ||
                      '',
                    ).toUpperCase() ===
                    'TERM'
                      ? 'settlement-term'
                      : 'settlement-immediate',
                  ]"
                >
                  {{
                    paymentSettlementLabel(
                      sale.payment_settlement,
                    )
                  }}
                </span>

                <small>
                  {{
                    paymentChannelLabel(
                      sale,
                    )
                  }}
                </small>
              </td>

              <td>
                <strong>
                  {{
                    formatDate(
                      sale.due_date,
                    )
                  }}
                </strong>

                <small>
                  {{
                    Number(
                      sale.payment_terms_days ||
                      0,
                    )
                  }}
                  hari
                </small>
              </td>

              <td class="money-cell">
                {{
                  formatCurrency(
                    sale.tax_base_amount,
                  )
                }}
              </td>

              <td class="money-cell">
                {{
                  formatCurrency(
                    sale.tax_amount,
                  )
                }}
              </td>

              <td class="money-cell total-cell">
                {{
                  formatCurrency(
                    sale.total,
                  )
                }}
              </td>

              <td>
                <span
                  :class="[
                    'print-badge',
                    sale.is_printed
                      ? 'print-done'
                      : 'print-pending',
                  ]"
                >
                  {{
                    printStateLabel(
                      sale,
                    )
                  }}
                </span>

                <small>
                  {{
                    printAuditLabel(
                      sale,
                    )
                  }}
                </small>
              </td>

              <td>
                <button
                  type="button"
                  class="view-invoice-button"
                  @click="
                    openInvoice(
                      sale,
                    )
                  "
                >
                  Lihat Faktur
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <footer
        v-if="
          !loading &&
          Number(
            meta.total ||
            0,
          ) > 0
        "
        class="pagination-row"
      >
        <p>
          Menampilkan
          {{
            formatNumber(
              Number(
                meta.offset ||
                0,
              ) +
              1,
            )
          }}
          –
          {{
            formatNumber(
              Math.min(
                Number(
                  meta.offset ||
                  0,
                ) +
                  sales.length,
                Number(
                  meta.total ||
                  0,
                ),
              ),
            )
          }}
          dari
          {{
            formatNumber(
              meta.total,
            )
          }}
        </p>

        <div class="pagination-actions">
          <button
            type="button"
            class="page-button"
            :disabled="
              !canGoPrevious
            "
            @click="previousPage"
          >
            ← Sebelumnya
          </button>

          <span>
            {{ currentPage }}
          </span>

          <button
            type="button"
            class="page-button"
            :disabled="
              !canGoNext
            "
            @click="nextPage"
          >
            Berikutnya →
          </button>
        </div>
      </footer>
    </section>


    <section class="customer-tax-panel">
      <div class="customer-tax-heading">
        <div>
          <p class="section-kicker">
            Customer Master
          </p>

          <h2>
            Data Customer & Pajak
          </h2>

          <p>
            Lengkapi identitas toko, alamat, NPWP/NIK, PKP, TKU dan tempo default
            tanpa membuat customer baru.
          </p>
        </div>

        <div class="customer-heading-actions">
          <button
            type="button"
            class="customer-add-button"
            :disabled="
              customerLoading ||
              customerSaving
            "
            @click="
              openCreateCustomer
            "
          >
            <span>＋</span>
            Tambah Toko
          </button>

          <button
            type="button"
            class="secondary-button"
            :disabled="
              customerLoading ||
              customerExporting
            "
            @click="
              printCustomerReport
            "
          >
            Print / PDF
          </button>

          <button
            type="button"
            class="primary-button"
            :disabled="
              customerLoading ||
              customerExporting
            "
            @click="
              exportCustomersExcel
            "
          >
            Export Excel
          </button>
        </div>
      </div>

      <div class="customer-summary-grid">
        <article>
          <span>
            Customer Aktif
          </span>

          <strong>
            {{
              formatNumber(
                customerSummary.active,
              )
            }}
          </strong>
        </article>

        <article>
          <span>
            Data Pajak Lengkap
          </span>

          <strong>
            {{
              formatNumber(
                customerSummary.complete,
              )
            }}
          </strong>
        </article>

        <article>
          <span>
            Perlu Dilengkapi
          </span>

          <strong class="warning-number">
            {{
              formatNumber(
                customerSummary.incomplete,
              )
            }}
          </strong>
        </article>

        <article>
          <span>
            Customer PKP
          </span>

          <strong>
            {{
              formatNumber(
                customerSummary.pkp,
              )
            }}
          </strong>
        </article>
      </div>

      <div class="customer-filter-grid">
        <div class="search-shell">
          <svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
          >
            <circle
              cx="11"
              cy="11"
              r="7"
            />
            <path
              d="m20 20-3.5-3.5"
            />
          </svg>

          <input
            v-model="
              customerSearch
            "
            type="search"
            placeholder="Cari nama toko, kode, telepon, NPWP/NIK, alamat, TKU..."
          />

          <button
            v-if="
              customerSearch
            "
            type="button"
            class="search-clear"
            aria-label="Hapus pencarian"
            @click="
              customerSearch = ''
            "
          >
            <FasIcon name="close" />
          </button>
        </div>

        <FasSelect
          v-model="
            customerTaxFilter
          "
          class="finance-select"
        >
          <option value="ALL">
            Semua Data Pajak
          </option>

          <option value="COMPLETE">
            Data Pajak Lengkap
          </option>

          <option value="INCOMPLETE">
            Perlu Dilengkapi
          </option>

          <option value="PKP">
            PKP
          </option>

          <option value="NON_PKP">
            Non-PKP
          </option>
        </FasSelect>

        <FasSelect
          v-model="
            customerStatusFilter
          "
          class="finance-select"
        >
          <option value="ACTIVE">
            Customer Aktif
          </option>

          <option value="ALL">
            Semua Status
          </option>

          <option value="INACTIVE">
            Customer Nonaktif
          </option>
        </FasSelect>
      </div>

      <div
        v-if="
          customerLoading
        "
        class="loading-state customer-loading-state"
      >
        <span class="loading-spinner"></span>

        <p>
          Memuat data customer...
        </p>
      </div>

      <div
        v-else-if="
          filteredCustomers.length ===
          0
        "
        class="empty-state customer-empty-state"
      >
        <h3>
          Customer tidak ditemukan
        </h3>

        <p>
          Ubah kata pencarian atau filter customer.
        </p>
      </div>

      <div
        v-else
        class="customer-table-scroll"
      >
        <table class="customer-tax-table">
          <thead>
            <tr>
              <th>
                Customer
              </th>

              <th>
                Kontak
              </th>

              <th>
                Alamat
              </th>

              <th>
                NPWP / NIK
              </th>

              <th>
                Nama Pajak
              </th>

              <th>
                PKP
              </th>

              <th>
                TKU
              </th>

              <th>
                Tempo Default
              </th>

              <th>
                Kelengkapan
              </th>

              <th>
                Aksi
              </th>
            </tr>
          </thead>

          <tbody>
            <tr
              v-for="
                customer in filteredCustomers
              "
              :key="
                customer.id
              "
            >
              <td>
                <strong>
                  {{
                    customer.name
                  }}
                </strong>

                <small>
                  {{
                    customer.customer_code
                  }}
                </small>

                <span
                  :class="[
                    'customer-active-badge',
                    customer.is_active
                      ? 'customer-active'
                      : 'customer-inactive',
                  ]"
                >
                  {{
                    customer.is_active
                      ? 'Aktif'
                      : 'Nonaktif'
                  }}
                </span>
              </td>

              <td>
                <strong>
                  {{
                    customer.phone ||
                    '—'
                  }}
                </strong>

                <small>
                  {{
                    customer.email ||
                    '—'
                  }}
                </small>
              </td>

              <td class="customer-address-cell">
                {{
                  customer.shipping_address ||
                  '—'
                }}
              </td>

              <td>
                <span class="tax-id">
                  {{
                    customerTaxIdentityLabel(
                      customer,
                    )
                  }}
                </span>
              </td>

              <td>
                <strong>
                  {{
                    customer.tax_name ||
                    '—'
                  }}
                </strong>

                <small>
                  {{
                    customer.tax_address ||
                    'Alamat pajak belum diisi'
                  }}
                </small>
              </td>

              <td>
                <span
                  :class="[
                    'pkp-badge',
                    customer.is_pkp
                      ? 'pkp-on'
                      : 'pkp-off',
                  ]"
                >
                  {{
                    customer.is_pkp
                      ? 'PKP'
                      : 'Non-PKP'
                  }}
                </span>
              </td>

              <td>
                {{
                  customer.tku_id ||
                  '—'
                }}
              </td>

              <td>
                <strong>
                  {{
                    Number(
                      customer.payment_terms_days ||
                      0,
                    )
                  }}
                  hari
                </strong>
              </td>

              <td>
                <span
                  :class="[
                    'tax-completeness-badge',
                    customerTaxComplete(
                      customer,
                    )
                      ? 'tax-complete'
                      : 'tax-incomplete',
                  ]"
                >
                  {{
                    customerTaxStatusLabel(
                      customer,
                    )
                  }}
                </span>
              </td>

              <td>
                <button
                  type="button"
                  class="customer-edit-button"
                  @click="
                    openCustomerEditor(
                      customer,
                    )
                  "
                >
                  Lengkapi / Edit
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <footer class="customer-table-footer">
        <span>
          Menampilkan
          {{
            formatNumber(
              filteredCustomers.length,
            )
          }}
          dari
          {{
            formatNumber(
              customers.length,
            )
          }}
          customer.
        </span>

        <span>
          Perubahan di sini memperbarui customer yang sama di sistem.
        </span>
      </footer>
    </section>


    <Teleport to="body">
      <Transition name="modal">
        <div
          v-if="
            customerModalOpen
          "
          class="customer-modal-shell"
        >
          <button
            type="button"
            class="customer-modal-backdrop"
            aria-label="Tutup form customer"
            @click="
              closeCustomerEditor
            "
          ></button>

          <section class="customer-modal" role="dialog" aria-modal="true" aria-label="Data customer">
            <header class="customer-modal-header">
              <div>
                <p>
                  {{
                    customerModalMode ===
                    'create'
                      ? 'Customer Master'
                      : 'Customer & Pajak'
                  }}
                </p>

                <h3>
                  {{
                    customerModalMode ===
                    'create'
                      ? 'Tambah Toko Baru'
                      : selectedCustomer
                          ?.name ||
                        'Customer'
                  }}
                </h3>

                <small>
                  {{
                    customerModalMode ===
                    'create'
                      ? 'Customer baru akan tersedia di seluruh sistem PT FAS.'
                      : selectedCustomer
                          ?.customer_code ||
                        ''
                  }}
                </small>
              </div>

              <button
                type="button"
                class="close-invoice-button"
                  aria-label="Tutup dialog"
                :disabled="
                  customerSaving
                "
                @click="
                  closeCustomerEditor
                "
              >
                <FasIcon name="close" />
              </button>
            </header>

            <form
              class="customer-modal-form"
              @submit.prevent="
                saveCustomer
              "
            >
              <div class="customer-form-body">
              <div
                v-if="
                  customerModalMode ===
                    'create' &&
                  customerDuplicateCandidates.length >
                    0
                "
                class="customer-duplicate-warning"
              >
                <div>
                  <strong>
                    Periksa customer yang mungkin sudah terdaftar
                  </strong>

                  <span>
                    Sistem mencocokkan nama toko, nomor telepon, dan NPWP/NIK.
                  </span>
                </div>

                <button
                  v-for="
                    duplicate in customerDuplicateCandidates
                  "
                  :key="
                    duplicate.id
                  "
                  type="button"
                  class="customer-duplicate-item"
                  @click="
                    openCustomerEditor(
                      duplicate,
                    )
                  "
                >
                  <strong>
                    {{
                      duplicate.name
                    }}
                  </strong>

                  <span>
                    {{
                      duplicate.customer_code
                    }}
                    ·
                    {{
                      duplicate.phone ||
                      'tanpa telepon'
                    }}
                    ·
                    {{
                      duplicate.tax_id_number ||
                      'tanpa NPWP/NIK'
                    }}
                  </span>
                </button>
              </div>

              <section class="customer-form-section">
                <div class="customer-form-heading">
                  <span>
                    Identitas Toko
                  </span>

                  <small>
                    Data operasional customer.
                  </small>
                </div>

                <div class="customer-form-grid">
                  <label>
                    <span>
                      Nama Toko
                    </span>

                    <input
                      v-model="
                        customerForm.name
                      "
                      type="text"
                      autocomplete="organization"
                      placeholder="Contoh: Apotek Sehat Jaya"
                      autofocus
                      required
                    />
                  </label>

                  <label>
                    <span>
                      Nomor Telepon
                    </span>

                    <input
                      v-model="
                        customerForm.phone
                      "
                      type="tel"
                      autocomplete="tel"
                      placeholder="Contoh: 081234567890"
                    />
                  </label>

                  <label>
                    <span>
                      Email
                    </span>

                    <input
                      v-model="
                        customerForm.email
                      "
                      type="email"
                      autocomplete="email"
                      placeholder="nama@toko.co.id"
                    />
                  </label>

                  <label>
                    <span>
                      Tempo Default
                    </span>

                    <div class="customer-number-field">
                      <input
                        v-model.number="
                          customerForm.payment_terms_days
                        "
                        type="number"
                        min="0"
                        max="365"
                      />

                      <strong>
                        HARI
                      </strong>
                    </div>
                  </label>

                  <label class="customer-span-two">
                    <span>
                      Alamat Toko / Pengiriman
                    </span>

                    <textarea
                      v-model="
                        customerForm.shipping_address
                      "
                      autocomplete="street-address"
                      placeholder="Jalan, nomor bangunan, kelurahan, kecamatan, kota dan kode pos"
                      rows="3"
                    ></textarea>
                  </label>
                </div>
              </section>

              <section class="customer-form-section">
                <div class="customer-form-heading">
                  <span>
                    Identitas Pajak
                  </span>

                  <small>
                    Dapat dilengkapi setelah toko menyerahkan NPWP/NIK.
                  </small>
                </div>

                <div class="customer-form-grid">
                  <label>
                    <span>
                      Jenis ID Pajak
                    </span>

                    <FasSelect
                      v-model="
                        customerForm.tax_id_type
                      "
                    >
                      <option value="">
                        Belum Ada
                      </option>

                      <option value="NPWP">
                        NPWP
                      </option>

                      <option value="NIK">
                        NIK
                      </option>
                    </FasSelect>
                  </label>

                  <label>
                    <span>
                      Nomor NPWP / NIK
                    </span>

                    <input
                      v-model="
                        customerForm.tax_id_number
                      "
                      type="text"
                      autocomplete="off"
                      placeholder="Nomor tanpa titik atau strip juga boleh"
                    />
                  </label>

                  <label>
                    <span>
                      Nama Wajib Pajak
                    </span>

                    <input
                      v-model="
                        customerForm.tax_name
                      "
                      type="text"
                    />
                  </label>

                  <label>
                    <span>
                      TKU
                    </span>

                    <input
                      v-model="
                        customerForm.tku_id
                      "
                      type="text"
                    />
                  </label>

                  <label class="customer-span-two">
                    <span>
                      Alamat Pajak
                    </span>

                    <textarea
                      v-model="
                        customerForm.tax_address
                      "
                      rows="3"
                    ></textarea>
                  </label>

                  <label class="customer-toggle-row">
                    <span>
                      Status PKP
                    </span>

                    <button
                      type="button"
                      :class="[
                        'customer-toggle',
                        customerForm.is_pkp
                          ? 'customer-toggle-on'
                          : '',
                      ]"
                      @click="
                        customerForm.is_pkp =
                          !customerForm.is_pkp
                      "
                    >
                      <span></span>

                      {{
                        customerForm.is_pkp
                          ? 'PKP'
                          : 'Non-PKP'
                      }}
                    </button>
                  </label>

                  <label class="customer-toggle-row">
                    <span>
                      Status Customer
                    </span>

                    <button
                      type="button"
                      :class="[
                        'customer-toggle',
                        customerForm.is_active
                          ? 'customer-toggle-on'
                          : '',
                      ]"
                      @click="
                        customerForm.is_active =
                          !customerForm.is_active
                      "
                    >
                      <span></span>

                      {{
                        customerForm.is_active
                          ? 'Aktif'
                          : 'Nonaktif'
                      }}
                    </button>
                  </label>
                </div>
              </section>

              </div>

              <footer class="customer-modal-actions">
                <button
                  type="button"
                  class="secondary-button"
                  :disabled="
                    customerSaving
                  "
                  @click="
                    closeCustomerEditor
                  "
                >
                  Batal
                </button>

                <button
                  type="submit"
                  class="primary-button"
                  :disabled="
                    customerSaving
                  "
                >
                  {{
                    customerSaving
                      ? 'Menyimpan...'
                      : customerModalMode ===
                          'create'
                        ? 'Tambahkan ke Customer Master'
                        : 'Simpan Data Customer'
                  }}
                </button>
              </footer>
            </form>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="modal">
        <div
          v-if="reportPreviewOpen"
          class="report-preview-shell"
        >
          <button
            type="button"
            class="report-preview-backdrop"
            aria-label="Tutup preview laporan"
            @click="closeSalesReportPreview"
          ></button>

          <section class="report-preview-modal">
            <header class="report-preview-toolbar">
              <div>
                <p>Finance Report</p>

                <h3>
                  Preview Rekap Penjualan
                </h3>

                <small>
                  {{ periodLabel }}
                </small>
              </div>

              <div class="report-preview-actions">
                <button
                  type="button"
                  class="secondary-button"
                  :disabled="
                    reportPreviewLoading ||
                    exporting
                  "
                  @click="exportExcel"
                >
                  Export Excel
                </button>

                <button
                  type="button"
                  class="primary-button"
                  :disabled="
                    reportPreviewLoading ||
                    exporting
                  "
                  @click="printSalesReport"
                >
                  Cetak / Simpan PDF
                </button>

                <button
                  type="button"
                  class="close-invoice-button"
                  aria-label="Tutup dialog"
                  @click="closeSalesReportPreview"
                >
                  <FasIcon name="close" />
                </button>
              </div>
            </header>

            <div
              v-if="reportPreviewLoading"
              class="report-preview-loading"
            >
              <span class="loading-spinner"></span>

              <p>
                Menyiapkan preview laporan...
              </p>
            </div>

            <div
              v-else
              class="report-paper-stage"
            >
              <article class="report-paper">
                <header class="report-paper-header">
                  <div>
                    <h2>
                      {{ companyProfile.name }}
                    </h2>

                    <h3>
                      REKAP PENJUALAN
                    </h3>
                  </div>

                  <div class="report-paper-period">
                    <span>Periode</span>
                    <strong>
                      {{ periodLabel }}
                    </strong>

                    <span>
                      Data:
                      {{
                        formatNumber(
                          reportPreviewRows.length,
                        )
                      }}
                      faktur
                    </span>
                  </div>
                </header>

                <section class="report-paper-summary">
                  <div>
                    <span>Total Penjualan</span>
                    <strong>
                      {{
                        formatCurrency(
                          summary.grand_total,
                        )
                      }}
                    </strong>
                  </div>

                  <div>
                    <span>Jumlah Faktur</span>
                    <strong>
                      {{
                        formatNumber(
                          summary.invoice_count,
                        )
                      }}
                    </strong>
                  </div>

                  <div>
                    <span>DPP</span>
                    <strong>
                      {{
                        formatCurrency(
                          summary.tax_base_amount,
                        )
                      }}
                    </strong>
                  </div>

                  <div>
                    <span>PPN</span>
                    <strong>
                      {{
                        formatCurrency(
                          summary.tax_amount,
                        )
                      }}
                    </strong>
                  </div>
                </section>

                <div class="report-paper-table-scroll">
                  <table class="report-paper-table">
                    <thead>
                      <tr>
                        <th>No</th>
                        <th>Tanggal</th>
                        <th>No. Faktur</th>
                        <th>No. SO</th>
                        <th>Customer</th>
                        <th>NPWP / NIK</th>
                        <th>Skema</th>
                        <th>Jatuh Tempo</th>
                        <th>DPP</th>
                        <th>PPN</th>
                        <th>Total</th>
                      </tr>
                    </thead>

                    <tbody>
                      <tr
                        v-for="(
                          sale,
                          index
                        ) in reportPreviewRows"
                        :key="
                          sale.id ||
                          index
                        "
                      >
                        <td>{{ index + 1 }}</td>

                        <td>
                          {{
                            formatDate(
                              sale.invoice_date,
                            )
                          }}
                        </td>

                        <td>
                          <strong>
                            {{
                              sale.invoice_number ||
                              '—'
                            }}
                          </strong>
                        </td>

                        <td>
                          {{
                            sale.order_number ||
                            '—'
                          }}
                        </td>

                        <td>
                          <strong>
                            {{
                              sale.store_name ||
                              '—'
                            }}
                          </strong>

                          <small>
                            {{
                              sale.customer_code ||
                              ''
                            }}
                          </small>
                        </td>

                        <td>
                          {{
                            sale.customer_tax_id_number ||
                            '—'
                          }}
                        </td>

                        <td>
                          {{
                            paymentSettlementLabel(
                              sale.payment_settlement,
                            )
                          }}
                        </td>

                        <td>
                          {{
                            formatDate(
                              sale.due_date,
                            )
                          }}
                        </td>

                        <td class="report-money">
                          {{
                            formatCurrency(
                              sale.tax_base_amount,
                            )
                          }}
                        </td>

                        <td class="report-money">
                          {{
                            formatCurrency(
                              sale.tax_amount,
                            )
                          }}
                        </td>

                        <td class="report-money report-total">
                          {{
                            formatCurrency(
                              sale.total,
                            )
                          }}
                        </td>
                      </tr>

                      <tr
                        v-if="
                          reportPreviewRows.length ===
                          0
                        "
                      >
                        <td
                          colspan="11"
                          class="report-empty-row"
                        >
                          Tidak ada data pada filter ini.
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>

                <footer class="report-paper-footer">
                  <span>
                    Dicetak dari Sistem PT FAS
                  </span>

                  <span>
                    {{
                      formatDateTime(
                        new Date(),
                      )
                    }}
                  </span>
                </footer>
              </article>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="modal">
        <div
          v-if="invoiceOpen"
          class="invoice-modal-shell"
        >
          <button
            type="button"
            class="invoice-backdrop"
            aria-label="Tutup faktur"
            @click="closeInvoice"
          ></button>

          <section class="invoice-modal">
            <header class="invoice-modal-toolbar">
              <div>
                <p>
                  Arsip Faktur
                </p>

                <h3>
                  {{
                    selectedInvoice
                      ?.invoice_number ||
                    'Memuat faktur...'
                  }}
                </h3>
              </div>

              <div class="invoice-toolbar-actions">
                <button
                  v-if="
                    selectedInvoice
                  "
                  type="button"
                  class="secondary-button"
                  :disabled="
                    invoicePrinting
                  "
                  @click="
                    printSelectedInvoice
                  "
                >
                  {{
                    invoicePrinting
                      ? 'Menyiapkan...'
                      : 'Cetak / PDF'
                  }}
                </button>

                <button
                  type="button"
                  class="close-invoice-button"
                  aria-label="Tutup dialog"
                  @click="closeInvoice"
                >
                  <FasIcon name="close" />
                </button>
              </div>
            </header>

            <div
              v-if="invoiceLoading"
              class="invoice-loading"
            >
              <span class="loading-spinner"></span>
              Memuat faktur...
            </div>

            <div
              v-else-if="
                selectedInvoice
              "
              class="invoice-original-preview-stage"
            >
              <article
                v-for="
                  page in invoicePrintPages
                "
                :key="
                  `${selectedInvoice.id}-${page.number}`
                "
                class="invoice-original-document"
              >
                <header class="invoice-original-header">
                  <div class="invoice-company-box">
                    <strong>
                      {{ invoiceIssuer.name }}
                    </strong>

                    <span
                      v-for="(
                        line,
                        lineIndex
                      ) in invoiceCompanyAddressLines"
                      :key="
                        `finance-company-address-${lineIndex}`
                      "
                    >
                      {{ line }}
                    </span>

                    <span>
                      No. Izin :
                      {{ invoiceIssuer.license_number }}
                    </span>

                    <span>
                      NPWP :
                      {{ invoiceIssuer.tax_id }}
                    </span>
                  </div>

                  <div class="invoice-middle-meta">
                    <div>
                      <span>
                        No. SJ
                      </span>

                      <strong>
                        {{
                          selectedInvoice
                            .delivery_note_number ||
                          '—'
                        }}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Tgl.
                      </span>

                      <strong>
                        {{
                          formatDate(
                            selectedInvoice
                              .invoice_date,
                          )
                        }}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Tgl J. Tempo
                      </span>

                      <strong>
                        {{
                          formatDate(
                            selectedInvoice
                              .due_date,
                          )
                        }}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Salesman
                      </span>

                      <strong>
                        {{
                          selectedInvoice
                            .sales_name ||
                          '—'
                        }}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Skema Bayar
                      </span>

                      <strong>
                        {{
                          invoicePaymentSettlementLabel(
                            selectedInvoice
                              .payment_settlement,
                            selectedInvoice
                              .payment_channel,
                          )
                        }}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Kanal Bayar
                      </span>

                      <strong>
                        {{
                          invoicePaymentChannelLabel(
                            selectedInvoice
                              .payment_channel,
                          )
                        }}
                      </strong>

                      <span
                        v-if="
                          selectedInvoice
                            .payment_provider
                        "
                      >
                        {{
                          selectedInvoice
                            .payment_provider
                        }}
                      </span>
                    </div>

                    <div>
                      <span>
                        Tempo
                      </span>

                      <strong>
                        {{
                          selectedInvoice
                            .payment_settlement ===
                          'TERM'
                            ? invoicePaymentTermsLabel(
                                selectedInvoice
                                  .payment_terms_days,
                              )
                            : 'LANGSUNG'
                        }}
                      </strong>
                    </div>

                    <div>
                      <span>
                        Gudang
                      </span>

                      <strong>
                        {{
                          invoiceWarehouse
                        }}
                      </strong>

                      <span
                        v-if="
                          selectedInvoice
                            .warehouse_classification
                        "
                      >
                        {{
                          selectedInvoice
                            .warehouse_classification
                        }}
                      </span>
                    </div>
                  </div>

                  <div class="invoice-customer-box">
                    <span>
                      Kepada Yth,
                    </span>

                    <strong>
                      {{
                        invoiceCustomerName
                      }}
                    </strong>

                    <span class="invoice-customer-address">
                      {{
                        invoiceCustomerAddress
                      }}
                    </span>

                    <span
                      v-if="
                        selectedInvoice
                          .customer_tax_id_number
                      "
                    >
                      NPWP/NIK :
                      {{
                        selectedInvoice
                          .customer_tax_id_number
                      }}
                    </span>

                    <span>
                      {{
                        selectedInvoice
                          .customer_code ||
                        ''
                      }}
                    </span>
                  </div>
                </header>

                <div class="invoice-reference-row">
                  <div>
                    <span>
                      No. Faktur
                    </span>

                    <strong>
                      {{
                        selectedInvoice
                          .invoice_number ||
                        '—'
                      }}
                    </strong>
                  </div>

                  <div>
                    <span>
                      No. Sales Order
                    </span>

                    <strong>
                      {{
                        selectedInvoice
                          .order_number ||
                        '—'
                      }}
                    </strong>
                  </div>

                  <div>
                    <span>
                      Halaman
                    </span>

                    <strong>
                      {{ page.number }}
                      /
                      {{
                        invoicePrintPageCount
                      }}
                    </strong>
                  </div>
                </div>

                <div class="invoice-original-table-scroll">
                  <table class="invoice-original-table">
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
                        v-for="
                          item in page.items
                        "
                        :key="
                          item.id ||
                          item.__originalIndex
                        "
                      >
                        <td>
                          {{
                            item.__originalIndex +
                            1
                          }}
                        </td>

                        <td class="invoice-original-product">
                          <strong>
                            {{
                              itemName(
                                item,
                              )
                            }}
                          </strong>

                          <small
                            v-if="
                              itemCode(
                                item,
                              )
                            "
                          >
                            {{
                              itemCode(
                                item,
                              )
                            }}
                          </small>
                        </td>

                        <td>
                          {{
                            item.batch_number ||
                            '—'
                          }}
                        </td>

                        <td>
                          {{
                            formatDate(
                              item.expiry_date,
                            )
                          }}
                        </td>

                        <td>
                          {{
                            itemQty(
                              item,
                            )
                          }}
                          {{
                            itemUnit(
                              item,
                            )
                          }}
                        </td>

                        <td>
                          {{
                            formatCurrency(
                              itemPrice(
                                item,
                              ),
                            )
                          }}
                        </td>

                        <td>
                          {{
                            itemDiscountPercent(
                              item,
                            )
                          }}
                        </td>

                        <td>
                          {{
                            formatCurrency(
                              itemDiscountAmount(
                                item,
                              ),
                            )
                          }}
                        </td>

                        <td>
                          {{
                            formatCurrency(
                              itemTotal(
                                item,
                              ),
                            )
                          }}
                        </td>
                      </tr>

                      <tr
                        v-if="
                          page.items.length ===
                          0
                        "
                      >
                        <td
                          colspan="9"
                          class="invoice-original-empty"
                        >
                          Detail item belum tersedia.
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>

                <div
                  v-if="
                    !page.isLast
                  "
                  class="invoice-continuation-footer"
                >
                  <span>
                    Bersambung ke halaman
                    {{
                      page.number +
                      1
                    }}
                  </span>

                  <strong>
                    {{
                      selectedInvoice
                        .invoice_number
                    }}
                  </strong>
                </div>

                <template
                  v-if="
                    page.isLast
                  "
                >
                  <section class="invoice-original-bottom-grid">
                    <div class="invoice-original-signature-box">
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

                    <div class="invoice-original-tax-box">
                      <div>
                        <span>
                          Total
                        </span>

                        <strong>
                          {{
                            formatCurrency(
                              selectedInvoice
                                .subtotal,
                            )
                          }}
                        </strong>
                      </div>

                      <div>
                        <span>
                          Discount
                        </span>

                        <strong>
                          {{
                            formatCurrency(
                              selectedInvoice
                                .discount_amount,
                            )
                          }}
                        </strong>
                      </div>

                      <div>
                        <span>
                          DPP
                          {{
                            selectedInvoice
                              .tax_mode ===
                            'STANDARD_PPN'
                              ? 'Nilai Lain'
                              : ''
                          }}
                        </span>

                        <strong>
                          {{
                            formatCurrency(
                              selectedInvoice
                                .tax_base_amount,
                            )
                          }}
                        </strong>
                      </div>

                      <div>
                        <span>
                          {{
                            invoiceTaxLabel
                          }}
                        </span>

                        <strong>
                          {{
                            formatCurrency(
                              selectedInvoice
                                .ppn_amount ??
                              selectedInvoice
                                .tax_amount,
                            )
                          }}
                        </strong>
                      </div>

                      <div class="invoice-original-tax-total">
                        <span>
                          Jumlah
                        </span>

                        <strong>
                          {{
                            formatCurrency(
                              selectedInvoice
                                .total,
                            )
                          }}
                        </strong>
                      </div>
                    </div>
                  </section>

                  <section class="invoice-original-footer">
                    <div class="invoice-original-terbilang">
                      <strong>
                        Nilai dalam Huruf :
                      </strong>

                      <span>
                        {{
                          invoiceTerbilang
                        }}
                      </span>
                    </div>
                  </section>
                </template>
              </article>
            </div>
          </section>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.finance-page {
  min-height: 100%;
  padding:
    1.5rem
    1.6rem
    2.5rem;
  color: var(--fas-text);
}

.finance-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 2rem;
}

.finance-kicker-row {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.finance-kicker {
  color: var(--fas-accent-text);
  font-size: 0.72rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.live-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.36rem;
  border:
    1px solid
    rgba(16, 185, 129, 0.22);
  border-radius: 999px;
  background:
    rgba(16, 185, 129, 0.08);
  padding:
    0.25rem
    0.5rem;
  color: #6ee7b7;
  font-size: 0.55rem;
  font-weight: 800;
  letter-spacing: 0.08em;
}

.live-dot {
  width: 0.42rem;
  height: 0.42rem;
  border-radius: 999px;
  background: #34d399;
  box-shadow:
    0 0 0 3px
    rgba(52, 211, 153, 0.08);
}

.finance-title {
  margin-top: 0.45rem;
  color: var(--fas-text);
  font-size: clamp(
    1.8rem,
    3vw,
    2.3rem
  );
  font-weight: 850;
  letter-spacing: -0.035em;
}

.finance-description {
  max-width: 46rem;
  margin-top: 0.6rem;
  color: var(--fas-muted);
  font-size: 0.8rem;
  line-height: 1.45rem;
}

.last-update {
  margin-top: 0.4rem;
  color: #64748b;
  font-size: 0.6rem;
}

.header-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.6rem;
}

.primary-button,
.secondary-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.48rem;
  min-height: 2.6rem;
  border-radius: 0.7rem;
  padding:
    0
    0.9rem;
  font-size: 0.66rem;
  font-weight: 800;
  transition:
    border-color 160ms ease,
    background 160ms ease,
    opacity 160ms ease,
    transform 160ms ease;
  cursor: pointer;
}

.primary-button svg,
.secondary-button svg {
  width: 0.95rem;
  height: 0.95rem;
}

.primary-button {
  border:
    1px solid
    rgba(59, 130, 246, 0.56);
  background: var(--fas-accent);
  color: #ffffff;
  box-shadow:
    0 8px 24px
    rgba(37, 99, 235, 0.18);
}

.secondary-button {
  border:
    1px solid
    var(--fas-border);
  background:
    rgba(15, 23, 42, 0.68);
  color: var(--fas-text-soft);
}

.primary-button:hover:not(:disabled),
.secondary-button:hover:not(:disabled) {
  transform: translateY(-1px);
}

.primary-button:disabled,
.secondary-button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.message {
  margin-top: 1rem;
  border-radius: 0.7rem;
  padding:
    0.75rem
    0.9rem;
  font-size: 0.68rem;
  font-weight: 700;
}

.message-error {
  border:
    1px solid
    rgba(248, 113, 113, 0.25);
  background:
    rgba(127, 29, 29, 0.15);
  color: #fca5a5;
}

.summary-grid {
  display: grid;
  grid-template-columns:
    repeat(
      4,
      minmax(0, 1fr)
    );
  gap: 0.9rem;
  margin-top: 1.4rem;
}

.summary-card {
  position: relative;
  min-height: 9.2rem;
  overflow: hidden;
  border:
    1px solid
    var(--fas-border);
  border-radius: 1rem;
  background:
    linear-gradient(
      145deg,
      rgba(15, 23, 42, 0.88),
      rgba(10, 17, 31, 0.94)
    );
  padding: 1rem;
}

.summary-main {
  border-color:
    rgba(59, 130, 246, 0.45);
  background:
    linear-gradient(
      145deg,
      rgba(30, 64, 175, 0.19),
      rgba(10, 17, 31, 0.96)
    );
}

.summary-icon {
  position: absolute;
  top: 0.85rem;
  right: 0.85rem;
  display: grid;
  width: 2.5rem;
  height: 2.5rem;
  place-items: center;
  border:
    1px solid
    rgba(59, 130, 246, 0.35);
  border-radius: 0.7rem;
  background:
    rgba(37, 99, 235, 0.1);
  color: var(--fas-accent-text);
}

.summary-icon svg {
  width: 1.1rem;
  height: 1.1rem;
}

.summary-label {
  color: var(--fas-muted);
  font-size: 0.66rem;
  font-weight: 700;
}

.summary-value {
  margin-top: 1.6rem;
  color: var(--fas-text);
  font-size: 1.45rem;
  font-weight: 850;
  letter-spacing: -0.03em;
}

.summary-detail {
  margin-top: 0.45rem;
  color: #64748b;
  font-size: 0.58rem;
  line-height: 1rem;
}

.analytics-panel,
.filters-panel,
.sales-panel {
  margin-top: 1rem;
  border:
    1px solid
    var(--fas-border);
  border-radius: 1rem;
  background:
    rgba(10, 17, 31, 0.72);
}

.analytics-panel {
  padding: 1rem;
}

.section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.section-kicker {
  color: var(--fas-accent-text);
  font-size: 0.56rem;
  font-weight: 800;
  letter-spacing: 0.09em;
  text-transform: uppercase;
}

.section-heading h2 {
  margin-top: 0.25rem;
  color: var(--fas-text);
  font-size: 0.95rem;
  font-weight: 800;
}

.period-chip {
  border:
    1px solid
    var(--fas-border);
  border-radius: 999px;
  background:
    rgba(15, 23, 42, 0.65);
  padding:
    0.42rem
    0.68rem;
  color: var(--fas-muted);
  font-size: 0.56rem;
  font-weight: 700;
}

.analytics-grid {
  display: grid;
  grid-template-columns:
    repeat(
      4,
      minmax(0, 1fr)
    );
  gap: 0.65rem;
  margin-top: 0.9rem;
}

.metric-box {
  border:
    1px solid
    rgba(51, 65, 85, 0.7);
  border-radius: 0.8rem;
  background:
    rgba(15, 23, 42, 0.48);
  padding: 0.8rem;
}

.metric-box span {
  display: block;
  color: var(--fas-muted);
  font-size: 0.58rem;
  font-weight: 700;
}

.metric-box strong {
  display: block;
  margin-top: 0.45rem;
  color: var(--fas-text);
  font-size: 0.82rem;
}

.metric-box small {
  display: block;
  margin-top: 0.3rem;
  color: #64748b;
  font-size: 0.52rem;
}

.ratio-grid {
  display: grid;
  grid-template-columns:
    repeat(
      3,
      minmax(0, 1fr)
    );
  gap: 0.65rem;
  margin-top: 0.65rem;
}

.ratio-card {
  border:
    1px solid
    rgba(51, 65, 85, 0.7);
  border-radius: 0.8rem;
  background:
    rgba(15, 23, 42, 0.35);
  padding: 0.75rem;
}

.ratio-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.8rem;
}

.ratio-heading div {
  display: grid;
  gap: 0.18rem;
}

.ratio-heading span {
  color: var(--fas-muted);
  font-size: 0.54rem;
}

.ratio-heading strong {
  color: var(--fas-text-soft);
  font-size: 0.61rem;
}

.ratio-heading b {
  color: var(--fas-accent-text);
  font-size: 0.68rem;
}

.ratio-track {
  height: 0.33rem;
  margin-top: 0.58rem;
  overflow: hidden;
  border-radius: 999px;
  background:
    rgba(51, 65, 85, 0.55);
}

.ratio-track span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background:
    linear-gradient(
      90deg,
      var(--fas-accent),
      var(--fas-accent-text)
    );
}

.filters-panel {
  padding: 0.85rem;
}

.filter-top-row {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    12rem;
  gap: 0.7rem;
}

.search-shell {
  position: relative;
  display: flex;
  align-items: center;
  min-height: 2.6rem;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.7rem;
  background:
    rgba(7, 14, 26, 0.82);
}

.search-shell > svg {
  width: 0.95rem;
  height: 0.95rem;
  margin-left: 0.75rem;
  color: #64748b;
  flex: 0 0 auto;
}

.search-shell input {
  width: 100%;
  border: 0;
  outline: none;
  background: transparent;
  padding:
    0.65rem
    2.2rem
    0.65rem
    0.6rem;
  color: var(--fas-text);
  font-size: 0.64rem;
}

.search-shell input::placeholder {
  color: #526176;
}

.search-clear {
  position: absolute;
  right: 0.55rem;
  display: grid;
  width: 1.55rem;
  height: 1.55rem;
  place-items: center;
  border: 0;
  border-radius: 0.45rem;
  background:
    rgba(51, 65, 85, 0.4);
  color: #94a3b8;
  cursor: pointer;
}

.finance-select {
  min-height: 2.6rem;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.7rem;
  outline: none;
  background:
    rgba(7, 14, 26, 0.82);
  padding:
    0
    0.75rem;
  color: var(--fas-text-soft);
  font-size: 0.62rem;
}

.preset-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 0.5rem;
  margin-top: 0.7rem;
}

.preset-button,
.apply-period-button {
  min-height: 2.2rem;
  border-radius: 0.6rem;
  padding:
    0
    0.7rem;
  font-size: 0.58rem;
  font-weight: 750;
  cursor: pointer;
}

.preset-button {
  border:
    1px solid
    var(--fas-border);
  background:
    rgba(15, 23, 42, 0.55);
  color: var(--fas-muted);
}

.preset-active {
  border-color:
    rgba(59, 130, 246, 0.5);
  background:
    rgba(37, 99, 235, 0.14);
  color: var(--fas-accent-text);
}

.custom-period {
  display: flex;
  align-items: flex-end;
  gap: 0.45rem;
  margin-left: auto;
}

.custom-period label {
  display: grid;
  gap: 0.26rem;
}

.custom-period label span {
  color: #64748b;
  font-size: 0.49rem;
  font-weight: 700;
}

.custom-period input {
  min-height: 2.2rem;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.6rem;
  outline: none;
  background:
    rgba(7, 14, 26, 0.82);
  padding:
    0
    0.62rem;
  color: var(--fas-text-soft);
  color-scheme: dark;
  font-size: 0.58rem;
}

.period-separator {
  display: flex;
  min-height: 2.2rem;
  align-items: center;
  color: #64748b;
  font-size: 0.65rem;
}

.apply-period-button {
  border:
    1px solid
    rgba(59, 130, 246, 0.36);
  background:
    rgba(37, 99, 235, 0.12);
  color: var(--fas-accent-text);
}

.sales-panel {
  overflow: hidden;
}

.sales-heading {
  padding:
    0.9rem
    1rem;
  border-bottom:
    1px solid
    var(--fas-border);
}

.table-description {
  margin-top: 0.28rem;
  color: #64748b;
  font-size: 0.55rem;
}

.table-status {
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.55rem;
  background:
    rgba(15, 23, 42, 0.55);
  padding:
    0.38rem
    0.55rem;
  color: #64748b;
  font-size: 0.52rem;
  font-weight: 700;
}

.loading-state,
.empty-state {
  display: grid;
  min-height: 18rem;
  place-items: center;
  align-content: center;
  gap: 0.7rem;
  padding: 2rem;
  text-align: center;
}

.loading-state p,
.empty-state p {
  max-width: 28rem;
  color: var(--fas-muted);
  font-size: 0.63rem;
  line-height: 1.2rem;
}

.empty-state h3 {
  color: var(--fas-text);
  font-size: 0.86rem;
}

.empty-icon {
  display: grid;
  width: 3rem;
  height: 3rem;
  place-items: center;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.8rem;
  background:
    rgba(15, 23, 42, 0.55);
  color: #64748b;
}

.empty-icon svg {
  width: 1.35rem;
  height: 1.35rem;
}

.loading-spinner {
  width: 1.35rem;
  height: 1.35rem;
  border:
    2px solid
    rgba(96, 165, 250, 0.15);
  border-top-color:
    var(--fas-accent-text);
  border-radius: 999px;
  animation:
    finance-spin
    0.75s
    linear
    infinite;
}

.table-scroll-shell {
  max-height: 35rem;
  overflow: auto;
}

.finance-table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
}

.finance-table thead {
  position: sticky;
  top: 0;
  z-index: 4;
}

.finance-table th {
  border-bottom:
    1px solid
    var(--fas-border);
  background: #0d1524;
  padding:
    0.65rem
    0.7rem;
  color: #748399;
  font-size: 0.49rem;
  font-weight: 800;
  letter-spacing: 0.055em;
  text-align: left;
  text-transform: uppercase;
  white-space: nowrap;
}

.finance-table td {
  border-bottom:
    1px solid
    rgba(51, 65, 85, 0.48);
  padding:
    0.7rem;
  color: var(--fas-text-soft);
  font-size: 0.58rem;
  vertical-align: top;
}

.finance-table tbody tr {
  background:
    rgba(8, 15, 27, 0.3);
  transition:
    background 150ms ease;
}

.finance-table tbody tr:hover {
  background:
    rgba(30, 64, 175, 0.08);
}

.finance-table td strong {
  display: block;
  color: var(--fas-text);
  font-size: 0.6rem;
  line-height: 0.95rem;
}

.finance-table td small {
  display: block;
  margin-top: 0.18rem;
  color: #64748b;
  font-size: 0.48rem;
  line-height: 0.75rem;
}

.invoice-number {
  color: var(--fas-accent-text) !important;
  font-family:
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    monospace;
}

.tax-id {
  display: inline-block;
  max-width: 8.5rem;
  overflow: hidden;
  color: #cbd5e1;
  font-family:
    ui-monospace,
    SFMono-Regular,
    Menlo,
    Monaco,
    Consolas,
    monospace;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.money-cell {
  text-align: right;
  white-space: nowrap;
}

.total-cell {
  color: #e2e8f0 !important;
  font-weight: 800;
}

.settlement-badge,
.print-badge {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding:
    0.25rem
    0.45rem;
  font-size: 0.48rem;
  font-weight: 800;
  white-space: nowrap;
}

.settlement-term {
  border:
    1px solid
    rgba(245, 158, 11, 0.25);
  background:
    rgba(245, 158, 11, 0.08);
  color: #fbbf24;
}

.settlement-immediate {
  border:
    1px solid
    rgba(16, 185, 129, 0.24);
  background:
    rgba(16, 185, 129, 0.08);
  color: #6ee7b7;
}

.print-done {
  border:
    1px solid
    rgba(59, 130, 246, 0.28);
  background:
    rgba(37, 99, 235, 0.09);
  color: var(--fas-accent-text);
}

.print-pending {
  border:
    1px solid
    rgba(148, 163, 184, 0.2);
  background:
    rgba(71, 85, 105, 0.12);
  color: #94a3b8;
}

.view-invoice-button {
  min-height: 2rem;
  border:
    1px solid
    rgba(59, 130, 246, 0.34);
  border-radius: 0.55rem;
  background:
    rgba(37, 99, 235, 0.1);
  padding:
    0
    0.65rem;
  color: var(--fas-accent-text);
  font-size: 0.52rem;
  font-weight: 800;
  white-space: nowrap;
  cursor: pointer;
}

.view-invoice-button:hover {
  background:
    rgba(37, 99, 235, 0.18);
}

.pagination-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-top:
    1px solid
    var(--fas-border);
  padding:
    0.75rem
    1rem;
}

.pagination-row p {
  color: #64748b;
  font-size: 0.55rem;
}

.pagination-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.pagination-actions > span {
  display: grid;
  min-width: 2rem;
  height: 2rem;
  place-items: center;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.5rem;
  color: var(--fas-text-soft);
  font-size: 0.56rem;
  font-weight: 800;
}

.page-button {
  min-height: 2rem;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.5rem;
  background:
    rgba(15, 23, 42, 0.55);
  padding:
    0
    0.62rem;
  color: var(--fas-muted);
  font-size: 0.52rem;
  font-weight: 750;
  cursor: pointer;
}

.page-button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.sales-heading-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 0.45rem;
}

.table-report-button,
.table-excel-button {
  min-height: 2rem;
  border-radius: 0.52rem;
  padding: 0 0.68rem;
  font-size: 0.51rem;
  font-weight: 800;
  cursor: pointer;
}

.table-report-button {
  border: 1px solid rgba(59, 130, 246, 0.34);
  background: rgba(37, 99, 235, 0.1);
  color: var(--fas-accent-text);
}

.table-excel-button {
  border: 1px solid rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.08);
  color: #6ee7b7;
}

.table-report-button:disabled,
.table-excel-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}


.message-success {
  border:
    1px solid
    rgba(16, 185, 129, 0.24);
  background:
    rgba(6, 78, 59, 0.16);
  color: #6ee7b7;
}

.customer-tax-panel {
  margin-top: 1rem;
  overflow: hidden;
  border:
    1px solid
    var(--fas-border);
  border-radius: 1rem;
  background:
    rgba(10, 17, 31, 0.72);
}

.customer-tax-heading {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  border-bottom:
    1px solid
    var(--fas-border);
  padding: 1rem;
}

.customer-tax-heading h2 {
  margin-top: 0.24rem;
  color: var(--fas-text);
  font-size: 1rem;
  font-weight: 850;
}

.customer-tax-heading p:not(.section-kicker) {
  max-width: 46rem;
  margin-top: 0.34rem;
  color: var(--fas-muted);
  font-size: 0.58rem;
  line-height: 1rem;
}

.customer-add-button {
  display: inline-flex;
  min-height: 2.6rem;
  align-items: center;
  justify-content: center;
  gap: 0.38rem;
  border:
    1px solid
    rgba(16, 185, 129, 0.35);
  border-radius: 0.7rem;
  background:
    rgba(16, 185, 129, 0.1);
  padding:
    0
    0.85rem;
  color: #6ee7b7;
  font-size: 0.62rem;
  font-weight: 850;
  cursor: pointer;
}

.customer-add-button span {
  font-size: 0.9rem;
  line-height: 1;
}

.customer-add-button:hover:not(:disabled) {
  background:
    rgba(16, 185, 129, 0.16);
}

.customer-add-button:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.customer-duplicate-warning {
  display: grid;
  gap: 0.5rem;
  margin-bottom: 0.85rem;
  border:
    1px solid
    rgba(245, 158, 11, 0.28);
  border-radius: 0.75rem;
  background:
    rgba(120, 53, 15, 0.12);
  padding: 0.7rem;
}

.customer-duplicate-warning > div {
  display: grid;
  gap: 0.18rem;
}

.customer-duplicate-warning > div strong {
  color: #fbbf24;
  font-size: 0.58rem;
}

.customer-duplicate-warning > div span {
  color: #94a3b8;
  font-size: 0.49rem;
}

.customer-duplicate-item {
  display: grid;
  gap: 0.12rem;
  border:
    1px solid
    rgba(245, 158, 11, 0.18);
  border-radius: 0.58rem;
  background:
    rgba(15, 23, 42, 0.5);
  padding:
    0.55rem
    0.62rem;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.customer-duplicate-item:hover {
  border-color:
    rgba(245, 158, 11, 0.38);
  background:
    rgba(245, 158, 11, 0.06);
}

.customer-duplicate-item strong {
  color: var(--fas-text);
  font-size: 0.55rem;
}

.customer-duplicate-item span {
  color: #94a3b8;
  font-size: 0.46rem;
}

.customer-heading-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.customer-summary-grid {
  display: grid;
  grid-template-columns:
    repeat(
      4,
      minmax(0, 1fr)
    );
  gap: 0.65rem;
  padding: 0.85rem 1rem 0;
}

.customer-summary-grid article {
  border:
    1px solid
    rgba(51, 65, 85, 0.68);
  border-radius: 0.75rem;
  background:
    rgba(15, 23, 42, 0.42);
  padding: 0.75rem;
}

.customer-summary-grid span {
  display: block;
  color: var(--fas-muted);
  font-size: 0.54rem;
  font-weight: 700;
}

.customer-summary-grid strong {
  display: block;
  margin-top: 0.34rem;
  color: var(--fas-text);
  font-size: 1rem;
  font-weight: 850;
}

.customer-summary-grid .warning-number {
  color: #fbbf24;
}

.customer-filter-grid {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    12rem
    11rem;
  gap: 0.65rem;
  padding: 0.85rem 1rem;
}

.customer-table-scroll {
  max-height: 34rem;
  overflow: auto;
  border-top:
    1px solid
    var(--fas-border);
  border-bottom:
    1px solid
    var(--fas-border);
}

.customer-tax-table {
  width: max-content;
  min-width: 100%;
  border-collapse: collapse;
}

.customer-tax-table thead {
  position: sticky;
  top: 0;
  z-index: 3;
}

.customer-tax-table th {
  border-bottom:
    1px solid
    var(--fas-border);
  background: #0d1524;
  padding:
    0.62rem
    0.7rem;
  color: #748399;
  font-size: 0.48rem;
  font-weight: 800;
  letter-spacing: 0.05em;
  text-align: left;
  text-transform: uppercase;
  white-space: nowrap;
}

.customer-tax-table td {
  max-width: 18rem;
  border-bottom:
    1px solid
    rgba(51, 65, 85, 0.46);
  padding:
    0.68rem
    0.7rem;
  color: var(--fas-text-soft);
  font-size: 0.57rem;
  line-height: 0.88rem;
  vertical-align: top;
}

.customer-tax-table td strong,
.customer-tax-table td small {
  display: block;
}

.customer-tax-table td strong {
  color: var(--fas-text);
  font-size: 0.6rem;
}

.customer-tax-table td small {
  margin-top: 0.16rem;
  color: #64748b;
  font-size: 0.47rem;
  line-height: 0.72rem;
}

.customer-address-cell {
  min-width: 13rem;
  white-space: pre-line;
}

.customer-active-badge,
.pkp-badge,
.tax-completeness-badge {
  display: inline-flex;
  align-items: center;
  margin-top: 0.28rem;
  border-radius: 999px;
  padding:
    0.22rem
    0.42rem;
  font-size: 0.44rem;
  font-weight: 800;
  white-space: nowrap;
}

.customer-active,
.pkp-on,
.tax-complete {
  border:
    1px solid
    rgba(16, 185, 129, 0.22);
  background:
    rgba(16, 185, 129, 0.08);
  color: #6ee7b7;
}

.customer-inactive,
.pkp-off {
  border:
    1px solid
    rgba(100, 116, 139, 0.22);
  background:
    rgba(71, 85, 105, 0.12);
  color: #94a3b8;
}

.tax-incomplete {
  border:
    1px solid
    rgba(245, 158, 11, 0.25);
  background:
    rgba(245, 158, 11, 0.08);
  color: #fbbf24;
}

.customer-edit-button {
  min-height: 2rem;
  border:
    1px solid
    rgba(59, 130, 246, 0.34);
  border-radius: 0.55rem;
  background:
    rgba(37, 99, 235, 0.1);
  padding:
    0
    0.65rem;
  color: var(--fas-accent-text);
  font-size: 0.5rem;
  font-weight: 800;
  white-space: nowrap;
  cursor: pointer;
}

.customer-table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding:
    0.72rem
    1rem;
  color: #64748b;
  font-size: 0.5rem;
}

.customer-modal-shell {
  position: fixed;
  z-index: 1300;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 1rem;
}

.customer-modal-backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background:
    rgba(2, 6, 23, 0.86);
  backdrop-filter:
    blur(5px);
}

.customer-modal {
  position: relative;
  z-index: 2;
  width: min(
    96vw,
    58rem
  );
  max-height: 94vh;
  overflow: auto;
  border:
    1px solid
    rgba(71, 85, 105, 0.78);
  border-radius: 1rem;
  background: #0b1220;
  box-shadow:
    0 28px 90px
    rgba(0, 0, 0, 0.5);
}

.customer-modal-header {
  position: sticky;
  top: 0;
  z-index: 4;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom:
    1px solid
    var(--fas-border);
  background:
    rgba(11, 18, 32, 0.98);
  padding:
    0.85rem
    0.95rem;
}

.customer-modal-header p {
  color: var(--fas-accent-text);
  font-size: 0.52rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.customer-modal-header h3 {
  margin-top: 0.16rem;
  color: var(--fas-text);
  font-size: 0.9rem;
}

.customer-modal-header small {
  display: block;
  margin-top: 0.14rem;
  color: #64748b;
  font-size: 0.5rem;
}

.customer-modal-form {
  padding: 0.95rem;
}

.customer-form-section + .customer-form-section {
  margin-top: 0.9rem;
  border-top:
    1px solid
    var(--fas-border);
  padding-top: 0.9rem;
}

.customer-form-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.65rem;
}

.customer-form-heading span {
  color: var(--fas-text);
  font-size: 0.7rem;
  font-weight: 800;
}

.customer-form-heading small {
  color: #64748b;
  font-size: 0.49rem;
}

.customer-form-grid {
  display: grid;
  grid-template-columns:
    repeat(
      2,
      minmax(0, 1fr)
    );
  gap: 0.7rem;
}

.customer-form-grid label {
  display: grid;
  gap: 0.32rem;
}

.customer-form-grid label > span {
  color: var(--fas-muted);
  font-size: 0.52rem;
  font-weight: 750;
}

.customer-form-grid input,
.customer-form-grid textarea,
.customer-form-grid select {
  width: 100%;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.62rem;
  outline: none;
  background:
    rgba(7, 14, 26, 0.82);
  padding:
    0.68rem
    0.72rem;
  color: var(--fas-text);
  font: inherit;
  font-size: 0.62rem;
}

.customer-form-grid textarea {
  min-height: 5.4rem;
  resize: vertical;
  line-height: 1rem;
}

.customer-form-grid select {
  color-scheme: dark;
}

.customer-span-two {
  grid-column: span 2;
}

.customer-number-field {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    auto;
}

.customer-number-field input {
  border-radius:
    0.62rem
    0
    0
    0.62rem;
}

.customer-number-field strong {
  display: flex;
  align-items: center;
  border:
    1px solid
    var(--fas-border);
  border-left: 0;
  border-radius:
    0
    0.62rem
    0.62rem
    0;
  background:
    rgba(15, 23, 42, 0.72);
  padding:
    0
    0.65rem;
  color: #94a3b8;
  font-size: 0.5rem;
}

.customer-toggle-row {
  align-content: start;
}

.customer-toggle {
  display: flex;
  min-height: 2.65rem;
  align-items: center;
  gap: 0.55rem;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.62rem;
  background:
    rgba(7, 14, 26, 0.82);
  padding:
    0
    0.7rem;
  color: #94a3b8;
  font-size: 0.58rem;
  font-weight: 800;
  cursor: pointer;
}

.customer-toggle > span {
  position: relative;
  width: 2.1rem;
  height: 1.12rem;
  border-radius: 999px;
  background:
    rgba(71, 85, 105, 0.72);
}

.customer-toggle > span::after {
  position: absolute;
  top: 0.14rem;
  left: 0.14rem;
  width: 0.84rem;
  height: 0.84rem;
  border-radius: 999px;
  background: #cbd5e1;
  content: '';
  transition:
    transform 160ms ease;
}

.customer-toggle-on {
  border-color:
    rgba(16, 185, 129, 0.3);
  color: #6ee7b7;
}

.customer-toggle-on > span {
  background:
    rgba(16, 185, 129, 0.38);
}

.customer-toggle-on > span::after {
  transform:
    translateX(0.98rem);
  background: #6ee7b7;
}

.customer-modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.55rem;
  margin-top: 1rem;
  border-top:
    1px solid
    var(--fas-border);
  padding-top: 0.85rem;
}

.customer-loading-state,
.customer-empty-state {
  border-top:
    1px solid
    var(--fas-border);
}

.report-preview-shell {
  position: fixed;
  z-index: 1250;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 1rem;
}

.report-preview-backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(2, 6, 23, 0.86);
  backdrop-filter: blur(5px);
}

.report-preview-modal {
  position: relative;
  z-index: 2;
  width: min(97vw, 94rem);
  max-height: 96vh;
  overflow: hidden;
  border: 1px solid rgba(71, 85, 105, 0.75);
  border-radius: 1rem;
  background: #0b1220;
  box-shadow: 0 28px 90px rgba(0, 0, 0, 0.5);
}

.report-preview-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid var(--fas-border);
  background: rgba(11, 18, 32, 0.98);
  padding: 0.8rem 0.9rem;
}

.report-preview-toolbar p {
  color: var(--fas-accent-text);
  font-size: 0.52rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.report-preview-toolbar h3 {
  margin-top: 0.16rem;
  color: var(--fas-text);
  font-size: 0.82rem;
}

.report-preview-toolbar small {
  display: block;
  margin-top: 0.18rem;
  color: #64748b;
  font-size: 0.5rem;
}

.report-preview-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.report-preview-loading {
  display: grid;
  min-height: 32rem;
  place-items: center;
  align-content: center;
  gap: 0.75rem;
  color: var(--fas-muted);
  font-size: 0.65rem;
}

.report-paper-stage {
  max-height: calc(96vh - 5rem);
  overflow: auto;
  background: #1e293b;
  padding: 1.2rem;
}

.report-paper {
  width: min(100%, 70rem);
  min-height: 42rem;
  margin: 0 auto;
  background: #ffffff;
  box-shadow: 0 18px 45px rgba(0, 0, 0, 0.28);
  padding: 1.1rem;
  color: #111827;
  font-family: Arial, Helvetica, sans-serif;
}

.report-paper-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 2px solid #111827;
  padding-bottom: 0.65rem;
}

.report-paper-header h2 {
  margin: 0;
  font-size: 1rem;
}

.report-paper-header h3 {
  margin: 0.22rem 0 0;
  font-size: 0.72rem;
  font-weight: 700;
}

.report-paper-period {
  display: grid;
  gap: 0.16rem;
  text-align: right;
}

.report-paper-period span {
  color: #64748b;
  font-size: 0.52rem;
}

.report-paper-period strong {
  font-size: 0.62rem;
}

.report-paper-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.45rem;
  margin-top: 0.7rem;
}

.report-paper-summary > div {
  border: 1px solid #cbd5e1;
  border-radius: 0.35rem;
  padding: 0.48rem;
}

.report-paper-summary span {
  display: block;
  color: #64748b;
  font-size: 0.45rem;
  text-transform: uppercase;
}

.report-paper-summary strong {
  display: block;
  margin-top: 0.22rem;
  font-size: 0.64rem;
}

.report-paper-table-scroll {
  margin-top: 0.7rem;
  overflow-x: auto;
}

.report-paper-table {
  width: 100%;
  min-width: 60rem;
  border-collapse: collapse;
}

.report-paper-table th,
.report-paper-table td {
  border: 1px solid #94a3b8;
  padding: 0.32rem 0.36rem;
  font-size: 0.47rem;
  line-height: 0.7rem;
  vertical-align: top;
}

.report-paper-table th {
  background: #f1f5f9;
  font-size: 0.43rem;
  text-transform: uppercase;
}

.report-paper-table td strong,
.report-paper-table td small {
  display: block;
}

.report-paper-table td small {
  margin-top: 0.12rem;
  color: #64748b;
  font-size: 0.4rem;
}

.report-money {
  text-align: right;
  white-space: nowrap;
}

.report-total {
  font-weight: 800;
}

.report-empty-row {
  padding: 1rem !important;
  color: #64748b;
  text-align: center;
}

.report-paper-footer {
  display: flex;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.65rem;
  border-top: 1px solid #cbd5e1;
  padding-top: 0.42rem;
  color: #64748b;
  font-size: 0.42rem;
}

.invoice-modal-shell {
  position: fixed;
  z-index: 1200;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 1.2rem;
}

.invoice-backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background:
    rgba(2, 6, 23, 0.82);
  backdrop-filter:
    blur(5px);
}

.invoice-modal {
  position: relative;
  z-index: 2;
  width: min(
    96vw,
    80rem
  );
  max-height: 94vh;
  overflow: auto;
  border:
    1px solid
    rgba(71, 85, 105, 0.75);
  border-radius: 1rem;
  background: #0b1220;
  box-shadow:
    0 28px 80px
    rgba(0, 0, 0, 0.42);
}

.invoice-modal-toolbar {
  position: sticky;
  top: 0;
  z-index: 4;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom:
    1px solid
    var(--fas-border);
  background:
    rgba(11, 18, 32, 0.96);
  padding:
    0.8rem
    0.9rem;
  backdrop-filter:
    blur(10px);
}

.invoice-modal-toolbar p {
  color: var(--fas-accent-text);
  font-size: 0.52rem;
  font-weight: 800;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.invoice-modal-toolbar h3 {
  margin-top: 0.18rem;
  color: var(--fas-text);
  font-size: 0.82rem;
}

.invoice-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.close-invoice-button {
  display: grid;
  width: 2.35rem;
  height: 2.35rem;
  place-items: center;
  border:
    1px solid
    var(--fas-border);
  border-radius: 0.6rem;
  background:
    rgba(15, 23, 42, 0.65);
  color: #94a3b8;
  font-size: 1.2rem;
  cursor: pointer;
}

.invoice-loading {
  display: flex;
  min-height: 22rem;
  align-items: center;
  justify-content: center;
  gap: 0.7rem;
  color: var(--fas-muted);
  font-size: 0.68rem;
}

.invoice-original-preview-stage {
  display: grid;
  gap: 1rem;
  padding: 1rem;
  background: #1e293b;
}

.invoice-original-document {
  width: min(100%, 74rem);
  margin: 0 auto;
  border: 1px solid #b6c1cd;
  background: #fffef6;
  padding: 0.7rem 0.75rem;
  box-shadow:
    0 18px 60px
    rgba(15, 23, 42, 0.18);
  color: #1f2937;
  font-family:
    "Courier New",
    Courier,
    monospace;
}

.invoice-original-header {
  display: grid;
  grid-template-columns:
    1.15fr
    0.9fr
    1.15fr;
  gap: 0.35rem;
}

.invoice-company-box,
.invoice-customer-box {
  display: flex;
  min-height: 5.1rem;
  flex-direction: column;
  gap: 0.1rem;
  border: 1px solid #708090;
  padding: 0.32rem 0.42rem;
  font-size: 0.53rem;
  line-height: 0.78rem;
}

.invoice-company-box strong,
.invoice-customer-box strong {
  font-size: 0.63rem;
}

.invoice-customer-address {
  white-space: pre-line;
}

.invoice-middle-meta {
  display: grid;
  grid-template-columns:
    0.78fr
    1.22fr;
  align-content: start;
  gap:
    0.08rem
    0.3rem;
  padding:
    0.22rem
    0.1rem;
  font-size: 0.51rem;
  line-height: 0.68rem;
}

.invoice-middle-meta > div {
  display: contents;
}

.invoice-middle-meta span {
  color: #64748b;
}

.invoice-middle-meta strong {
  color: #111827;
  font-size: 0.52rem;
}

.invoice-reference-row {
  display: grid;
  grid-template-columns:
    1fr
    1fr
    1fr;
  margin-top: 0.25rem;
}

.invoice-reference-row > div {
  min-height: 2.25rem;
  border: 1px solid #708090;
  padding: 0.22rem 0.34rem;
}

.invoice-reference-row > div + div {
  border-left: 0;
}

.invoice-reference-row span {
  display: block;
  color: #64748b;
  font-size: 0.45rem;
}

.invoice-reference-row strong {
  display: block;
  margin-top: 0.1rem;
  color: #111827;
  font-size: 0.54rem;
}

.invoice-original-table-scroll {
  overflow-x: auto;
}

.invoice-original-table {
  width: 100%;
  min-width: 61rem;
  border-collapse: collapse;
  table-layout: fixed;
}

.invoice-original-table th,
.invoice-original-table td {
  border: 1px solid #708090;
  padding: 0.28rem 0.32rem;
  font-size: 0.48rem;
  line-height: 0.68rem;
  vertical-align: top;
}

.invoice-original-table th {
  background: #eef2f7;
  font-size: 0.43rem;
  text-align: center;
  text-transform: uppercase;
}

.invoice-original-table th:nth-child(1),
.invoice-original-table td:nth-child(1) {
  width: 6%;
  text-align: center;
}

.invoice-original-table th:nth-child(2),
.invoice-original-table td:nth-child(2) {
  width: 27%;
}

.invoice-original-table th:nth-child(3),
.invoice-original-table td:nth-child(3) {
  width: 10%;
}

.invoice-original-table th:nth-child(4),
.invoice-original-table td:nth-child(4) {
  width: 8%;
}

.invoice-original-table th:nth-child(5),
.invoice-original-table td:nth-child(5) {
  width: 11%;
}

.invoice-original-table th:nth-child(6),
.invoice-original-table td:nth-child(6) {
  width: 12%;
}

.invoice-original-table th:nth-child(7),
.invoice-original-table td:nth-child(7) {
  width: 8%;
}

.invoice-original-table th:nth-child(8),
.invoice-original-table td:nth-child(8) {
  width: 9%;
}

.invoice-original-table th:nth-child(9),
.invoice-original-table td:nth-child(9) {
  width: 12%;
}

.invoice-original-product strong,
.invoice-original-product small {
  display: block;
}

.invoice-original-product small {
  margin-top: 0.08rem;
  color: #64748b;
  font-size: 0.39rem;
}

.invoice-original-empty {
  padding: 1rem !important;
  color: #64748b;
  text-align: center;
}

.invoice-continuation-footer {
  display: flex;
  justify-content: space-between;
  border:
    1px solid
    #708090;
  border-top: 0;
  padding:
    0.28rem
    0.38rem;
  font-size: 0.47rem;
}

.invoice-original-bottom-grid {
  display: grid;
  grid-template-columns:
    minmax(0, 1fr)
    22rem;
  margin-top: 0.25rem;
}

.invoice-original-signature-box {
  display: grid;
  min-height: 4.7rem;
  grid-template-columns:
    1fr
    1fr;
  border: 1px solid #708090;
}

.invoice-original-signature-box > div {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding:
    0.32rem
    0.42rem;
  font-size: 0.47rem;
  text-align: center;
}

.invoice-original-signature-box > div:first-child {
  border-right:
    1px solid
    #708090;
}

.invoice-original-signature-box span {
  color: #64748b;
}

.invoice-original-signature-box strong {
  font-size: 0.48rem;
}

.invoice-original-tax-box {
  border:
    1px solid
    #708090;
  border-left: 0;
}

.invoice-original-tax-box > div {
  display: flex;
  min-height: 0.93rem;
  align-items: center;
  justify-content: space-between;
  gap: 0.7rem;
  border-bottom:
    1px solid
    #94a3b8;
  padding:
    0.15rem
    0.34rem;
  font-size: 0.47rem;
}

.invoice-original-tax-box > div:last-child {
  border-bottom: 0;
}

.invoice-original-tax-box strong {
  white-space: nowrap;
}

.invoice-original-tax-total {
  font-weight: 850;
}

.invoice-original-footer {
  min-height: 3.3rem;
  border:
    1px solid
    #708090;
  border-top: 0;
}

.invoice-original-terbilang {
  display: flex;
  gap: 0.6rem;
  padding:
    0.44rem
    0.48rem;
  font-size: 0.46rem;
  line-height: 0.76rem;
}

.invoice-original-terbilang strong {
  white-space: nowrap;
}

.invoice-original-note {
  border:
    1px solid
    #708090;
  border-top: 0;
  padding:
    0.26rem
    0.4rem;
  font-size: 0.43rem;
}

.modal-enter-active,
.modal-leave-active {
  transition:
    opacity 160ms ease;
}

.modal-enter-active .invoice-modal,
.modal-leave-active .invoice-modal {
  transition:
    transform 180ms ease,
    opacity 180ms ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .invoice-modal,
.modal-leave-to .invoice-modal {
  opacity: 0;
  transform:
    translateY(0.7rem)
    scale(0.985);
}

@keyframes finance-spin {
  to {
    transform:
      rotate(360deg);
  }
}

@media (max-width: 1180px) {
  .summary-grid,
  .analytics-grid {
    grid-template-columns:
      repeat(
        2,
        minmax(0, 1fr)
      );
  }

  .ratio-grid {
    grid-template-columns:
      1fr;
  }

  .custom-period {
    width: 100%;
    margin-left: 0;
  }
}

@media (max-width: 760px) {
  .finance-page {
    padding:
      1rem
      0.9rem
      2rem;
  }

  .finance-header {
    align-items: stretch;
    flex-direction: column;
  }

  .header-actions {
    justify-content: flex-start;
  }

  .summary-grid,
  .analytics-grid {
    grid-template-columns:
      1fr;
  }

  .filter-top-row {
    grid-template-columns:
      1fr;
  }

  .custom-period {
    align-items: stretch;
    flex-direction: column;
  }

  .period-separator {
    display: none;
  }

  .section-heading,
  .pagination-row {
    align-items: flex-start;
    flex-direction: column;
  }

  .invoice-head-grid,
  .invoice-reference-grid,
  .invoice-bottom-grid {
    grid-template-columns:
      1fr;
  }

  .invoice-total-box {
    border-left:
      1px solid
      #64748b;
  }
}

@media (max-width: 900px) {
  .sales-heading-actions,
  .report-preview-toolbar,
  .report-preview-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .report-preview-toolbar {
    align-items: flex-start;
  }

  .report-paper-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}


@media (max-width: 820px) {
  .invoice-original-header {
    grid-template-columns: 1fr;
  }

  .invoice-reference-row {
    grid-template-columns: 1fr;
  }

  .invoice-reference-row > div + div {
    border-left:
      1px solid
      #708090;
    border-top: 0;
  }

  .invoice-original-bottom-grid {
    grid-template-columns: 1fr;
  }

  .invoice-original-tax-box {
    border-left:
      1px solid
      #708090;
    border-top: 0;
  }
}


@media (max-width: 1100px) {
  .customer-summary-grid {
    grid-template-columns:
      repeat(
        2,
        minmax(0, 1fr)
      );
  }

  .customer-filter-grid {
    grid-template-columns:
      1fr
      1fr;
  }

  .customer-filter-grid .search-shell {
    grid-column: span 2;
  }
}

@media (max-width: 720px) {
  .customer-tax-heading,
  .customer-table-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .customer-heading-actions {
    flex-wrap: wrap;
  }

  .customer-summary-grid,
  .customer-filter-grid,
  .customer-form-grid {
    grid-template-columns:
      1fr;
  }

  .customer-filter-grid .search-shell,
  .customer-span-two {
    grid-column: span 1;
  }
}



.finance-section-nav {
  display:
    flex;
  flex-wrap:
    wrap;
  gap:
    0.45rem;
  margin-top:
    1rem;
  margin-bottom:
    1rem;
}

.finance-section-link {
  display:
    inline-flex;
  min-height:
    2.15rem;
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
    rgba(
      15,
      23,
      42,
      0.55
    );
  padding:
    0
    0.7rem;
  color:
    var(--fas-muted);
  font-size:
    0.52rem;
  font-weight:
    800;
  text-decoration:
    none;
  transition:
    border-color
      160ms
      ease,
    background
      160ms
      ease,
    color
      160ms
      ease;
}

.finance-section-link:hover {
  border-color:
    rgba(
      59,
      130,
      246,
      0.34
    );
  color:
    var(--fas-accent-text);
}

.finance-section-link-active {
  border-color:
    rgba(
      59,
      130,
      246,
      0.5
    );
  background:
    rgba(
      37,
      99,
      235,
      0.14
    );
  color:
    var(--fas-accent-text);
}


.finance-customers-page
.summary-grid,
.finance-customers-page
.analytics-panel,
.finance-customers-page
.filters-panel,
.finance-customers-page
.sales-panel,
.finance-customers-page
.header-actions {
  display:
    none !important;
}


.customer-modal {
  display: flex;
  width: min(56rem, 100%);
  max-height: calc(100dvh - 2rem);
  flex-direction: column;
  overflow: hidden;
  border-color: var(--fas-border-strong);
  background: var(--fas-card);
  box-shadow: var(--fas-shadow-card);
  color: var(--fas-text);
  color-scheme: inherit;
}

.customer-modal-header {
  position: static;
  flex-shrink: 0;
  background: var(--fas-card);
  padding: 1rem 1.25rem;
}

.customer-modal-header p {
  color: var(--fas-accent-text);
  font-size: 0.65rem;
}

.customer-modal-header h3 {
  margin-top: 0.3rem;
  font-size: 1.2rem;
  font-weight: 750;
}

.customer-modal-header small {
  color: var(--fas-muted);
  font-size: 0.75rem;
  line-height: 1.4;
}

.customer-modal-form {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  padding: 0;
}

.customer-form-body {
  min-height: 0;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 1.25rem;
}

.customer-form-section + .customer-form-section {
  margin-top: 1.25rem;
  padding-top: 1.25rem;
}

.customer-form-heading {
  flex-wrap: wrap;
  gap: 0.35rem;
  margin-bottom: 0.9rem;
}

.customer-form-heading span {
  font-size: 0.9rem;
}

.customer-form-heading small {
  color: var(--fas-muted);
  font-size: 0.72rem;
  line-height: 1.4;
}

.customer-form-grid {
  gap: 1rem;
}

.customer-form-grid label {
  min-width: 0;
  gap: 0.45rem;
}

.customer-form-grid label > span {
  color: var(--fas-text-soft);
  font-size: 0.75rem;
}

.customer-form-grid input,
.customer-form-grid textarea {
  min-height: 2.85rem;
  border-color: var(--fas-border-strong);
  border-radius: 0.85rem;
  background: var(--fas-input);
  padding: 0.7rem 0.85rem;
  color: var(--fas-text);
  font-size: 0.85rem;
  line-height: 1.5;
  transition: border-color 160ms ease, box-shadow 160ms ease;
}

.customer-form-grid textarea {
  min-height: 5rem;
}

.customer-form-grid input::placeholder,
.customer-form-grid textarea::placeholder {
  color: var(--fas-muted);
  opacity: 0.85;
}

.customer-form-grid input:focus,
.customer-form-grid textarea:focus {
  outline: none;
  border-color: var(--fas-accent-border);
  box-shadow: 0 0 0 3px var(--fas-accent-soft);
}

.customer-number-field input {
  border-radius: 0.85rem 0 0 0.85rem;
}

.customer-number-field strong {
  border-color: var(--fas-border-strong);
  border-radius: 0 0.85rem 0.85rem 0;
  background: var(--fas-elevated);
  color: var(--fas-muted);
  font-size: 0.65rem;
}

.customer-toggle {
  min-height: 2.85rem;
  border-color: var(--fas-border-strong);
  border-radius: 0.85rem;
  background: var(--fas-input);
  color: var(--fas-text-soft);
  font-size: 0.8rem;
}

.customer-toggle-on {
  border-color: var(--fas-accent-border);
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}

.customer-toggle-on > span {
  background: var(--fas-accent);
}

.customer-toggle-on > span::after {
  background: #ffffff;
}

.customer-modal-actions {
  flex-shrink: 0;
  margin-top: 0;
  background: var(--fas-card);
  padding: 0.9rem 1.25rem;
}

.customer-modal-actions button {
  min-height: 2.85rem;
  font-size: 0.8rem;
}

.customer-modal .close-invoice-button {
  display: inline-flex;
  width: 2.5rem;
  height: 2.5rem;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid var(--fas-border-strong);
  border-radius: 0.8rem;
  background: var(--fas-elevated);
  color: var(--fas-text-soft);
}

@media (max-width: 640px) {
  .customer-modal-shell {
    padding: 0.6rem;
  }

  .customer-modal {
    max-height: calc(100dvh - 1.2rem);
  }

  .customer-modal-header,
  .customer-form-body {
    padding: 1rem;
  }

  .customer-form-grid {
    grid-template-columns: 1fr;
  }

  .customer-span-two {
    grid-column: auto;
  }

  .customer-modal-actions {
    flex-direction: column-reverse;
    padding: 0.85rem 1rem;
  }

  .customer-modal-actions button {
    width: 100%;
  }
}

</style>
