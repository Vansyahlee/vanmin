<script setup>
import { computed, onMounted, reactive, ref } from "vue";
import { useModalDialog } from "@/composables/useModalDialog";
import {
  getFinanceReportPeriods,
  getFinanceSales,
  getFinanceExpenses,
  getFinanceTaxRegister,
  getFinanceCash,
  getFinanceInvoiceByID,
} from "@/services/finance";
import { companyProfile, loadCompanyProfile } from "@/services/company";
import { financeToday, financeError } from "@/composables/useFinanceLedger";
import {
  monthDates,
  periodProfit,
  reportMoney as money,
  reportCents,
  reportDecimal,
} from "@/utils/financeReporting";
import {
  downloadFinanceXlsx,
  openFinancePrint,
  writeFinancePrint,
} from "@/utils/financeDocuments";
import { writeExpenseReceipt } from "@/utils/expenseReceipt";
import { expenseDetailsRows, expenseKindLabel } from "@/utils/expenseForms";
import "@/assets/finance-ledger.css";

const definitions = {
  SUMMARY: {
    label: "Laporan keuangan",
    short: "Ringkasan hasil usaha",
    icon: "M4 20V4h16v16z M8 8h8 M8 12h8 M8 16h5",
    note: "Kesimpulan penjualan, laba, dan arus kas tercatat.",
    headings: [
      "Bulan",
      "Penjualan sebelum PPN",
      "Laba kotor",
      "Laba bersih manajerial",
      "Kas masuk",
      "Kas keluar",
      "Dasar biaya",
    ],
    map: (x) => [
      x.period,
      money(x.revenue),
      money(periodProfit(x)?.gross),
      money(periodProfit(x)?.net),
      money(x.incoming),
      money(x.outgoing),
      periodProfit(x) ? "Lengkap" : "Belum lengkap / bukan bulan penuh",
    ],
  },
  EXPENSES: {
    label: "Laporan pengeluaran",
    short: "Rekap biaya perusahaan",
    icon: "M4 5h16v15H4z M4 9h16 M8 14h8 M12 11v6",
    note: "Total pengeluaran beserta penerima, referensi, dan saldo buku.",
    fetch: getFinanceExpenses,
    headings: [
      "Nomor",
      "Tanggal",
      "Jenis",
      "Penerima",
      "Uraian",
      "Nominal",
      "Akun",
      "Saldo sebelum",
      "Saldo sesudah",
      "Status",
    ],
    map: (x) => [
      x.expense_number,
      day(x.expense_date),
      expenseKindLabel(x.expense_kind),
      x.payee,
      x.description,
      money(x.amount),
      x.cash_account || x.method,
      balance(x.balance_before),
      balance(x.balance_after),
      x.status,
    ],
  },
  RECEIPTS: {
    label: "Struk & slip gaji",
    short: "Bukti pencatatan internal",
    icon: "M6 3h12v18l-3-2-3 2-3-2-3 2z M9 7h6 M9 11h6",
    note: "Lihat struk pengeluaran, pembayaran supplier, atau slip gaji.",
    fetch: getFinanceExpenses,
    headings: [
      "Nomor bukti",
      "Tanggal",
      "Jenis",
      "Penerima",
      "Nominal",
      "Referensi",
      "Status",
    ],
    map: (x) => [
      x.expense_number,
      day(x.expense_date),
      expenseKindLabel(x.expense_kind),
      x.payee,
      money(x.amount),
      x.reference_number || "—",
      x.status,
    ],
  },
  INVOICES: {
    label: "Faktur penjualan",
    short: "Dokumen transaksi",
    icon: "M5 3h10l4 4v14H5z M14 3v5h5 M8 12h8 M8 16h6",
    note: "Buka faktur lengkap tanpa berpindah ke form penjualan.",
    fetch: getFinanceSales,
    headings: [
      "Nomor faktur",
      "Tanggal",
      "Customer",
      "Sebelum PPN",
      "PPN",
      "Total faktur",
      "Status",
    ],
    map: (x) => [
      x.invoice_number,
      day(x.invoice_date),
      x.store_name,
      money(
        reportDecimal(reportCents(x.total) - reportCents(x.tax_amount || 0)),
      ),
      money(x.tax_amount || 0),
      money(x.total),
      x.status,
    ],
  },
  TAX: {
    label: "Laporan pajak",
    short: "Rekap sumber perpajakan",
    icon: "M12 3l9 5H3z M5 10v7 M10 10v7 M15 10v7 M20 10v7 M3 21h18",
    note: "Rekap DPP dan PPN dari faktur untuk pemeriksaan petugas.",
    fetch: getFinanceTaxRegister,
    headings: [
      "Faktur",
      "Tanggal",
      "Customer",
      "NPWP",
      "Skema",
      "Sebelum PPN",
      "DPP PPN",
      "PPN",
    ],
    map: (x) => [
      x.invoice_number,
      day(x.invoice_date),
      x.store_name,
      x.buyer_tin || "—",
      x.tax_mode,
      money(x.net_sales),
      money(x.tax_base),
      money(x.vat),
    ],
  },
  CASH: {
    label: "Laporan kas & bank",
    short: "Ringkasan pergerakan dana",
    icon: "M3 5h18v15H3z M3 9h18 M16 13h5v4h-5z",
    note: "Mutasi dan saldo buku untuk penelusuran penerimaan serta pengeluaran.",
    fetch: getFinanceCash,
    headings: [
      "Tanggal",
      "Akun",
      "Arah",
      "Jenis",
      "Pihak",
      "Referensi",
      "Nominal",
      "Saldo berjalan",
    ],
    map: (x) => [
      day(x.occurred_at),
      x.account,
      x.direction,
      x.kind,
      x.party,
      x.reference_number,
      money(x.amount),
      balance(x.running_balance),
    ],
  },
};
const day = (value) => {
  if (!value) return "—";
  const text = String(value);
  if (/^\d{4}-\d{2}-\d{2}$/.test(text)) return text;
  const date = new Date(text);
  if (Number.isNaN(date.getTime())) return "—";
  const parts = new Intl.DateTimeFormat("en-CA", {
    timeZone: "Asia/Jakarta",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).formatToParts(date);
  const get = (type) => parts.find((x) => x.type === type)?.value;
  return `${get("year")}-${get("month")}-${get("day")}`;
};
const balance = (value) =>
  value == null || value === "" ? "Belum tersedia" : money(value);
const form = reactive({ ...monthDates(financeToday().slice(0, 7)), query: "" });
const kind = ref("SUMMARY"),
  applied = ref(null),
  rows = ref([]),
  loading = ref(false),
  error = ref(""),
  page = ref(0),
  document = ref(null),
  inspecting = ref(false);
useModalDialog(
  () => (document.value ? ".reports-dialog" : ""),
  () => {
    document.value = null;
  },
);
const definition = computed(
  () => definitions[applied.value?.kind || kind.value],
);
const mapped = computed(() => rows.value.map(definition.value.map));
const visible = computed(() =>
  rows.value.slice(page.value * 10, page.value * 10 + 10),
);
const pages = computed(() => Math.max(1, Math.ceil(rows.value.length / 10)));
const period = computed(() =>
  applied.value
    ? `${applied.value.date_from} s.d. ${applied.value.date_to}`
    : "",
);
const sum = (items, field) =>
  reportDecimal(items.reduce((n, x) => n + reportCents(x[field] || 0), 0n));
const conclusion = computed(() => {
  if (!applied.value || !rows.value.length)
    return "Belum ada dokumen untuk periode dan pencarian ini.";
  const data = rows.value;
  switch (applied.value.kind) {
    case "SUMMARY": {
      const complete = data.every((x) => periodProfit(x));
      const profit = complete
        ? money(
            reportDecimal(
              data.reduce((n, x) => n + reportCents(periodProfit(x).net), 0n),
            ),
          )
        : "belum dapat disimpulkan karena dasar biaya belum lengkap atau rentang bukan bulan penuh";
      return `Penjualan sebelum PPN ${money(sum(data, "revenue"))}. Laba bersih manajerial ${profit}. Kas masuk tercatat ${money(sum(data, "incoming"))} dan kas keluar ${money(sum(data, "outgoing"))}. Penjualan dan laba bukan saldo kas.`;
    }
    case "EXPENSES":
    case "RECEIPTS":
      return `${data.length} bukti pengeluaran tercatat dengan nominal ${money(sum(data, "amount"))}. Bukti ini adalah pencatatan internal, bukan bukti transfer bank. Saldo mengikuti urutan mutasi buku per akun.`;
    case "INVOICES":
      return `${data.length} faktur dengan total ${money(sum(data, "total"))} termasuk PPN ${money(sum(data, "tax_amount"))}. Total faktur bukan jumlah pembayaran yang telah diterima.`;
    case "TAX":
      return `${data.length} sumber faktur: DPP ${money(sum(data, "tax_base"))} dan PPN ${money(sum(data, "vat"))}. Rekap internal untuk pemeriksaan; bukan bukti pelaporan atau persetujuan DJP/Coretax.`;
    default:
      return `${data.length} mutasi tercatat. Mutasi masuk ${money(
        sum(
          data.filter((x) => x.direction === "IN"),
          "amount",
        ),
      )}, keluar ${money(
        sum(
          data.filter((x) => x.direction === "OUT"),
          "amount",
        ),
      )}. Saldo buku bukan saldo bank terverifikasi.`;
  }
});
const select = async (value) => {
  if (loading.value || inspecting.value) return;
  kind.value = value;
  await load();
};
const preset = async (value) => {
  const y = financeToday().slice(0, 4);
  Object.assign(
    form,
    value === "year"
      ? { date_from: y + "-01-01", date_to: y + "-12-31" }
      : monthDates(financeToday().slice(0, 7)),
  );
  await load();
};
const load = async () => {
  if (loading.value) return;
  loading.value = true;
  error.value = "";
  applied.value = null;
  rows.value = [];
  page.value = 0;
  document.value = null;
  const params = { ...form, kind: kind.value };
  if (params.kind === "SUMMARY") params.query = "";
  try {
    if (
      !params.date_from ||
      !params.date_to ||
      params.date_from > params.date_to
    )
      throw new Error("Pilih rentang tanggal yang valid.");
    if (!companyProfile.loaded) await loadCompanyProfile();
    const data = [];
    if (params.kind === "SUMMARY") {
      const response = await getFinanceReportPeriods(params);
      if (!Array.isArray(response.data))
        throw new Error("Format laporan keuangan tidak sesuai.");
      data.push(...response.data);
    } else {
      let expected = null;
      const seen = new Set();
      for (let offset = 0; ; offset += 200) {
        const response = await definitions[params.kind].fetch({
          ...params,
          limit: 200,
          offset,
          status: ["EXPENSES", "RECEIPTS"].includes(params.kind)
            ? "POSTED"
            : "",
        });
        if (!Array.isArray(response.data))
          throw new Error("Format dokumen tidak sesuai.");
        const total =
          response.meta?.total == null ? null : Number(response.meta.total);
        if (total !== null && (!Number.isSafeInteger(total) || total < 0))
          throw new Error("Jumlah dokumen tidak valid.");
        if (offset === 0) expected = total;
        if (total !== expected)
          throw new Error("Data berubah saat diambil. Muat ulang laporan.");
        for (const row of response.data) {
          const id = row.id || row.invoice_id;
          if (id && seen.has(id))
            throw new Error("Urutan dokumen berubah. Muat ulang laporan.");
          if (id) seen.add(id);
          data.push(row);
        }
        if (
          expected !== null
            ? data.length >= expected
            : response.data.length < 200
        )
          break;
        if (!response.data.length || offset > 1000000)
          throw new Error("Dokumen belum termuat lengkap.");
      }
    }
    const query = params.query.trim().toLowerCase();
    rows.value =
      params.kind === "SUMMARY"
        ? data
        : data.filter(
            (x) =>
              !query ||
              definitions[params.kind]
                .map(x)
                .join(" ")
                .toLowerCase()
                .includes(query),
          );
    applied.value = params;
  } catch (e) {
    error.value = financeError(e);
    rows.value = [];
  } finally {
    loading.value = false;
  }
};
const reportTitle = computed(() => `${definition.value.label} — PT FAS`);
const description = computed(
  () =>
    `Periode ${period.value}. Pencarian: ${applied.value?.query || "Semua"}. ${rows.value.length} hasil. ${conclusion.value}`,
);
const exportReport = (format) => {
  try {
    if (!applied.value || !rows.value.length) return;
    if (format === "excel")
      downloadFinanceXlsx(
        `PTFAS_${applied.value.kind}_${applied.value.date_from}_${applied.value.date_to}.xlsx`,
        [
          [reportTitle.value],
          [description.value],
          definition.value.headings,
          ...mapped.value,
        ],
      );
    else
      writeFinancePrint(
        openFinancePrint(),
        reportTitle.value,
        description.value,
        definition.value.headings,
        mapped.value,
      );
  } catch (e) {
    error.value = financeError(e);
  }
};
const receiptRows = (x) => [
  ["Nomor bukti", x.expense_number],
  ["Tanggal", day(x.expense_date)],
  ["Jenis", expenseKindLabel(x.expense_kind)],
  ["Penerima", x.payee],
  ["Uraian", x.description],
  ...expenseDetailsRows(x, money),
  ["Nominal", money(x.amount)],
  ["Akun", x.cash_account || x.method],
  ["Saldo sebelum", balance(x.balance_before)],
  ["Saldo sesudah", balance(x.balance_after)],
  ["Referensi", x.reference_number || "—"],
  ["Dicatat oleh", x.created_by_name || "—"],
  ["Status", x.status],
];
const inspect = async (row) => {
  if (inspecting.value) return;
  error.value = "";
  inspecting.value = true;
  try {
    if (applied.value.kind === "INVOICES") {
      const response = await getFinanceInvoiceByID(row.id);
      const invoice = response?.data || response;
      if (!invoice?.invoice_number || !Array.isArray(invoice.items))
        throw new Error("Detail faktur belum lengkap.");
      document.value = { type: "invoice", value: invoice };
    } else document.value = { type: "receipt", value: row };
  } catch (e) {
    error.value = financeError(e);
  } finally {
    inspecting.value = false;
  }
};
const escape = (value) =>
  String(value ?? "").replace(
    /[&<>"']/g,
    (c) =>
      ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[
        c
      ],
  );
const invoiceHTML = computed(() => {
  if (document.value?.type !== "invoice") return "";
  const x = document.value.value,
    issuer = x.issuer_profile || companyProfile,
    items = x.items;
  const parts = Array.from(
    { length: Math.max(1, Math.ceil(items.length / 10)) },
    (_, i) => items.slice(i * 10, i * 10 + 10),
  );
  return parts
    .map(
      (items, index) =>
        `<article class="report-invoice"><header><div><h2>${escape(issuer.name)}</h2><p>${escape(issuer.address)}</p><p>NPWP: ${escape(issuer.tax_id || "—")} · Izin: ${escape(issuer.license_number || "—")}</p></div><div><h2>FAKTUR PENJUALAN</h2><p>${escape(x.invoice_number)}</p><p>${escape(day(x.invoice_date))} · Halaman ${index + 1}/${parts.length}</p></div></header><div class="invoice-recipient"><div><strong>${escape(x.store_name)}</strong><p>${escape(x.customer_tax_address || x.shipping_address || "—")}</p><p>NPWP/NIK: ${escape(x.customer_tax_id_number || "—")}</p></div><div><p>Order: ${escape(x.order_number || "—")}</p><p>Surat jalan: ${escape(x.delivery_note_number || "—")}</p><p>Tempo: ${escape(x.payment_terms_days)} hari · Jatuh tempo: ${escape(day(x.due_date))}</p></div></div><table><thead><tr>${["No", "Kode", "Nama produk", "Batch / ED", "Qty", "Unit", "Harga", "Diskon", "Jumlah"].map((h) => `<th>${h}</th>`).join("")}</tr></thead><tbody>${items.map((item, i) => `<tr>${[index * 10 + i + 1, item.product_code, item.product_name, `${item.batch_number || "—"} / ${day(item.expiry_date)}`, item.qty, item.unit, money(item.price), money(item.discount_amount || 0), money(item.subtotal)].map((v) => `<td>${escape(v)}</td>`).join("")}</tr>`).join("")}</tbody></table><footer><div><p>Penerima</p><br><p>(________________)</p></div><div><p>Hormat kami</p><br><p>(________________)</p></div><div><p>Subtotal: ${escape(money(x.subtotal))}</p><p>Diskon: ${escape(money(x.discount_amount || 0))}</p><p>PPN: ${escape(money(x.tax_amount || 0))}</p><strong>Total faktur: ${escape(money(x.total))}</strong></div></footer></article>`,
    )
    .join("");
});
const invoiceCSS = `@page{size:9.5in 5.5in;margin:5mm}*{box-sizing:border-box}body{margin:0;font:10px/1.35 Arial;color:#172033;background:white}.report-invoice{width:9.05in;min-height:5.05in;padding:3mm;background:white;color:#172033;break-after:page}.report-invoice:last-child{break-after:auto}.report-invoice header,.invoice-recipient,.report-invoice footer{display:flex;justify-content:space-between;gap:20px}.report-invoice header{border-bottom:2px solid #172033;padding-bottom:8px}.report-invoice h2{font-size:15px;margin:0 0 5px}.report-invoice p{margin:3px 0;white-space:pre-line;overflow-wrap:anywhere}.report-invoice header>div{max-width:55%}.report-invoice header>div:last-child{text-align:right}.invoice-recipient{margin:10px 0}.report-invoice table{width:100%;border-collapse:collapse;table-layout:fixed}.report-invoice th,.report-invoice td{font-size:9px;border:1px solid #cbd5e1;padding:4px;overflow-wrap:anywhere}.report-invoice th:nth-child(3){width:22%}.report-invoice th:first-child{width:4%}.report-invoice footer{margin-top:14px;break-inside:avoid}.report-invoice footer>div:last-child{text-align:right}`;
const exportDocument = (format) => {
  try {
    const d = document.value;
    if (!d) return;
    if (d.type === "receipt") {
      if (format === "excel")
        downloadFinanceXlsx(`Struk_${d.value.expense_number}.xlsx`, [
          [companyProfile.name],
          ["Bukti pencatatan internal, bukan bukti transfer bank"],
          ...receiptRows(d.value),
        ]);
      else
        writeExpenseReceipt(openFinancePrint(), d.value, receiptRows(d.value));
    } else if (format === "excel") {
      const x = d.value,
        issuer = x.issuer_profile || companyProfile;
      downloadFinanceXlsx(`Faktur_${x.invoice_number}.xlsx`, [
        [issuer.name],
        [issuer.address],
        [x.invoice_number, day(x.invoice_date), x.store_name],
        [
          "Kode",
          "Produk",
          "Batch",
          "ED",
          "Qty",
          "Unit",
          "Harga",
          "Diskon",
          "Jumlah",
        ],
        ...x.items.map((i) => [
          i.product_code,
          i.product_name,
          i.batch_number,
          day(i.expiry_date),
          i.qty,
          i.unit,
          money(i.price),
          money(i.discount_amount || 0),
          money(i.subtotal),
        ]),
        ["Subtotal", money(x.subtotal)],
        ["Diskon", money(x.discount_amount || 0)],
        ["PPN", money(x.tax_amount || 0)],
        ["Total", money(x.total)],
      ]);
    } else {
      const popup = openFinancePrint();
      popup.document.open();
      popup.onload = () => {
        popup.focus();
        popup.print();
      };
      popup.document.write(
        `<!doctype html><html lang="id"><head><meta charset="UTF-8"><title>${escape(d.value.invoice_number)}</title><style>${invoiceCSS}</style></head><body>${invoiceHTML.value}</body></html>`,
      );
      popup.document.close();
    }
  } catch (e) {
    error.value = financeError(e);
  }
};
onMounted(load);
</script>

<template>
  <main class="finance-ledger reports-hub">
    <header class="reports-heading">
      <div>
        <p class="eyebrow">COMPANY · REPORTS</p>
        <h1>Hasil kerja, dalam satu laporan.</h1>
        <p class="muted">
          Ringkasan perusahaan dan dokumen siap dibaca, dicetak, atau dibawa ke
          rapat.
        </p>
      </div>
      <span class="reports-mark" aria-hidden="true">↗</span>
    </header>
    <section class="report-categories" aria-label="Pilih laporan">
      <button
        v-for="(item, key) in definitions"
        :key="key"
        type="button"
        :class="['report-category', { active: kind === key }]"
        :aria-pressed="kind === key"
        :disabled="loading || inspecting"
        @click="select(key)"
      >
        <span class="category-icon"
          ><svg
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path :d="item.icon" /></svg></span
        ><strong>{{ item.label }}</strong
        ><small>{{ item.short }}</small>
      </button>
    </section>
    <form class="panel report-period" @submit.prevent="load">
      <div class="filters">
        <label
          >Dari tanggal<input
            v-model="form.date_from"
            type="date"
            required
            :disabled="loading" /></label
        ><label
          >Sampai tanggal<input
            v-model="form.date_to"
            type="date"
            required
            :disabled="loading" /></label
        ><label
          >Cari dokumen<input
            v-model="form.query"
            placeholder="Nomor, penerima, atau customer"
            :disabled="loading || kind === 'SUMMARY'"
        /></label>
      </div>
      <div class="actions">
        <button type="submit" class="primary" :disabled="loading">
          {{ loading ? "Menyiapkan laporan…" : "Tampilkan hasil" }}</button
        ><button type="button" :disabled="loading" @click="preset('month')">
          Bulan ini</button
        ><button type="button" :disabled="loading" @click="preset('year')">
          Tahun ini
        </button>
      </div>
    </form>
    <p v-if="error" role="alert" class="error">{{ error }}</p>
    <div v-if="loading" class="panel report-empty" role="status">
      Mengumpulkan dokumen dan menyusun ringkasan…
    </div>
    <template v-else-if="applied">
      <section class="report-conclusion">
        <span class="eyebrow">KESIMPULAN PERIODE</span>
        <h2>{{ definition.label }}</h2>
        <p class="period-caption">{{ period }}</p>
        <p>{{ conclusion }}</p>
      </section>
      <section class="panel report-paper">
        <header class="paper-top">
          <div>
            <span class="eyebrow">DOKUMEN LAPORAN</span>
            <h2>{{ definition.label }}</h2>
            <p class="muted">{{ definition.note }}</p>
          </div>
          <div class="actions">
            <button
              type="button"
              :disabled="!rows.length"
              @click="exportReport('print')"
            >
              Cetak / PDF</button
            ><button
              type="button"
              :disabled="!rows.length"
              @click="exportReport('excel')"
            >
              Export Excel
            </button>
          </div>
        </header>
        <div class="paper-letterhead">
          <div>
            <strong>{{
              companyProfile.name || "PT Furi Anugerah Sejahtera"
            }}</strong>
            <p>{{ companyProfile.address }}</p>
          </div>
          <div>
            <strong>{{ definition.label }}</strong>
            <p>Periode {{ period }}</p>
            <p>
              {{ rows.length }} hasil · {{ applied.query || "Semua dokumen" }}
            </p>
          </div>
        </div>
        <div v-if="!rows.length" class="report-empty">
          Belum ada hasil pada periode ini.
        </div>
        <div
          v-else-if="['RECEIPTS', 'INVOICES'].includes(applied.kind)"
          class="document-grid"
        >
          <article v-for="row in visible" :key="row.id" class="document-card">
            <span class="eyebrow">{{
              applied.kind === "INVOICES"
                ? "FAKTUR PENJUALAN"
                : expenseKindLabel(row.expense_kind)
            }}</span>
            <h3>{{ row.invoice_number || row.expense_number }}</h3>
            <p>{{ row.store_name || row.payee }}</p>
            <p class="muted">{{ day(row.invoice_date || row.expense_date) }}</p>
            <strong class="document-amount">{{
              money(row.total ?? row.amount)
            }}</strong
            ><button type="button" :disabled="inspecting" @click="inspect(row)">
              {{ applied.kind === "INVOICES" ? "Lihat faktur" : "Lihat struk" }}
            </button>
          </article>
        </div>
        <div v-else class="report-table">
          <table>
            <thead>
              <tr>
                <th v-for="h in definition.headings" :key="h">{{ h }}</th>
                <th v-if="applied.kind === 'EXPENSES'">Dokumen</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="(row, index) in visible"
                :key="row.id || row.period || index"
              >
                <td v-for="(cell, i) in definition.map(row)" :key="i">
                  {{ cell }}
                </td>
                <td v-if="applied.kind === 'EXPENSES'">
                  <button
                    type="button"
                    :disabled="inspecting"
                    @click="inspect(row)"
                  >
                    Lihat struk
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
        <footer class="paper-bottom">
          <span class="muted"
            >10 hasil per halaman · Cetak dan Excel mencakup seluruh hasil
            filter.</span
          >
          <div class="actions">
            <button type="button" :disabled="page === 0" @click="page--">
              Sebelumnya</button
            ><span>{{ page + 1 }} / {{ pages }}</span
            ><button
              type="button"
              :disabled="page + 1 >= pages"
              @click="page++"
            >
              Berikutnya
            </button>
          </div>
        </footer>
      </section>
    </template>
    <Teleport to="body"
      ><div
        v-if="document"
        class="reports-dialog-overlay"
        @click.self="document = null"
        @keydown.esc="document = null"
      >
        <section
          role="dialog"
          aria-modal="true"
          aria-label="Pratinjau dokumen laporan"
          class="reports-dialog"
        >
          <header class="dialog-toolbar">
            <strong>{{
              document.type === "invoice"
                ? "Faktur penjualan"
                : "Struk pengeluaran"
            }}</strong>
            <div>
              <button type="button" @click="exportDocument('print')">
                Cetak / PDF</button
              ><button type="button" @click="exportDocument('excel')">
                Export Excel</button
              ><button
                type="button"
                aria-label="Tutup dokumen"
                autofocus
                @click="document = null"
              >
                ✕
              </button>
            </div>
          </header>
          <div class="dialog-scroll">
            <div
              v-if="document.type === 'invoice'"
              class="report-invoice-preview"
              v-html="invoiceHTML"
            ></div>
            <article v-else class="receipt-preview">
              <header>
                <strong>{{ companyProfile.name }}</strong>
                <p>{{ companyProfile.address }}</p>
                <h2>
                  {{
                    document.value.expense_kind === "SALARY"
                      ? "Slip pembayaran gaji"
                      : "Struk pengeluaran"
                  }}
                </h2>
              </header>
              <dl>
                <div
                  v-for="([label, value], i) in receiptRows(document.value)"
                  :key="i"
                >
                  <dt>{{ label }}</dt>
                  <dd>{{ value }}</dd>
                </div>
              </dl>
              <p>
                Bukti pencatatan internal, bukan bukti transfer bank. Saldo buku
                per akun mengikuti tanggal dan urutan mutasi; pencatatan mundur
                dapat mengubah saldo historis.
              </p>
            </article>
          </div>
        </section>
      </div></Teleport
    >
  </main>
</template>

<style scoped>
.report-paper {
  padding: 24px;
}
.report-period > .actions {
  padding: 0 18px 18px;
}
.report-invoice-preview :deep(th:nth-child(3)) {
  width: 22%;
}
.report-invoice-preview :deep(th:first-child) {
  width: 4%;
}
.reports-hub .error {
  padding: 15px;
  border: 1px solid var(--fas-danger);
  border-radius: 12px;
  color: var(--fas-danger);
  background: var(--fas-danger-soft);
}

.reports-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 28px;
}
.reports-heading h1 {
  font-size: clamp(26px, 3vw, 38px);
  letter-spacing: -0.045em;
  margin: 12px 0;
}
.reports-mark {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  border: 1px solid var(--fas-accent-border);
  border-radius: 22px;
  color: var(--fas-accent-text);
  background: var(--fas-accent-soft);
  font-size: 30px;
  flex-shrink: 0;
}
.report-categories {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin-bottom: 24px;
}
.reports-hub .report-category {
  display: flex;
  align-items: flex-start;
  flex-direction: column;
  text-align: left;
  padding: 22px;
  gap: 12px;
  border: 1px solid var(--fas-border);
  border-radius: 20px;
  background: var(--fas-card);
  color: var(--fas-text);
  transition:
    transform 0.2s,
    border-color 0.2s;
}
.reports-hub .report-category:hover {
  transform: translateY(-3px);
}
.reports-hub .report-category.active {
  border-color: var(--fas-accent);
  background: linear-gradient(140deg, var(--fas-accent-soft), var(--fas-card));
}
.report-category strong {
  font-size: 15px;
}
.report-category small {
  color: var(--fas-muted);
  font-size: 11px;
}
.category-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border-radius: 13px;
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}
.category-icon svg {
  width: 23px;
  height: 23px;
}
.report-period {
  margin-bottom: 22px;
}
.report-conclusion {
  padding: 25px 28px;
  border: 1px solid var(--fas-accent-border);
  border-radius: 22px;
  background: var(--fas-accent-soft);
  margin: 22px 0;
}
.report-conclusion h2 {
  font-size: 23px;
  margin: 10px 0;
}
.report-conclusion > p:last-child {
  line-height: 1.9;
  font-size: 14px;
  max-width: 1000px;
}
.period-caption {
  font-size: 12px;
  color: var(--fas-accent-text);
  margin-bottom: 14px;
}
.paper-top,
.paper-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}
.paper-top h2 {
  font-size: 22px;
  margin: 10px 0;
}
.paper-top p {
  font-size: 12px;
}
.paper-letterhead {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  padding: 25px 0 18px;
  margin-top: 18px;
  border-bottom: 2px solid var(--fas-accent-border);
  font-size: 12px;
}
.paper-letterhead > div:last-child {
  text-align: right;
}
.paper-letterhead p {
  white-space: pre-line;
  color: var(--fas-muted);
  margin-top: 6px;
  font-size: 11px;
}
.report-table {
  overflow: auto;
  margin: 20px 0;
}
.report-table table {
  width: 100%;
  border-collapse: collapse;
  white-space: nowrap;
  font-size: 12px;
}
.report-table th,
.report-table td {
  padding: 14px 12px;
  border-bottom: 1px solid var(--fas-border);
  text-align: left;
}
.report-table th {
  color: var(--fas-muted);
  font-size: 11px;
}
.paper-bottom {
  padding-top: 20px;
}
.paper-bottom > .muted {
  font-size: 11px;
}
.document-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  padding-top: 22px;
}
.document-card {
  border: 1px solid var(--fas-border);
  border-radius: 16px;
  padding: 22px;
  background: var(--fas-overlay);
  min-width: 0;
}
.document-card h3 {
  font-size: 17px;
  margin: 14px 0 9px;
  overflow-wrap: anywhere;
}
.document-card p {
  font-size: 12px;
  margin: 7px 0;
}
.document-amount {
  display: block;
  font-size: 23px;
  margin: 20px 0;
  color: var(--fas-accent-text);
}
.document-card button {
  width: 100%;
}
.report-empty {
  padding: 55px 20px;
  text-align: center;
  color: var(--fas-muted);
}
.reports-dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(5, 12, 25, 0.65);
  backdrop-filter: blur(7px);
  z-index: 100;
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 24px;
}
.reports-dialog {
  width: min(1100px, 100%);
  max-height: 90vh;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--fas-border);
  border-radius: 22px;
  background: var(--fas-card);
  color: var(--fas-text);
  overflow: hidden;
}
.dialog-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  padding: 18px;
  border-bottom: 1px solid var(--fas-border);
}
.dialog-toolbar > div {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.dialog-toolbar button {
  padding: 10px 14px;
  border: 1px solid var(--fas-border);
  border-radius: 10px;
  background: var(--fas-overlay);
  color: var(--fas-text);
  cursor: pointer;
}
.dialog-scroll {
  overflow: auto;
  padding: 25px;
  background: var(--fas-bg);
}
.receipt-preview {
  width: min(390px, 100%);
  margin: auto;
  background: var(--fas-card);
  padding: 28px;
  border: 1px solid var(--fas-border);
  box-shadow: 0 15px 35px var(--fas-shadow);
}
.receipt-preview header {
  text-align: center;
  border-bottom: 1px dashed var(--fas-border);
  padding-bottom: 20px;
}
.receipt-preview h2 {
  margin-top: 18px;
  font-size: 19px;
}
.receipt-preview p {
  font-size: 11px;
  line-height: 1.8;
  white-space: pre-line;
}
.receipt-preview dl > div {
  display: flex;
  justify-content: space-between;
  gap: 15px;
  padding: 10px 0;
  font-size: 11px;
}
.receipt-preview dt {
  color: var(--fas-muted);
}
.receipt-preview dd {
  text-align: right;
  max-width: 60%;
  overflow-wrap: anywhere;
}
.receipt-preview > p {
  border-top: 1px dashed var(--fas-border);
  padding-top: 18px;
  color: var(--fas-muted);
}
.report-invoice-preview :deep(.report-invoice) {
  width: 9.05in;
  min-height: 5.05in;
  background: white;
  color: #172033;
  padding: 5mm;
  margin: 0 auto 20px;
  font: 10px/1.35 Arial;
}
.report-invoice-preview :deep(header),
.report-invoice-preview :deep(.invoice-recipient),
.report-invoice-preview :deep(footer) {
  display: flex;
  justify-content: space-between;
  gap: 20px;
}
.report-invoice-preview :deep(header) {
  border-bottom: 2px solid #172033;
  padding-bottom: 8px;
}
.report-invoice-preview :deep(header > div) {
  max-width: 55%;
}
.report-invoice-preview :deep(h2) {
  font-size: 15px;
  margin: 0 0 5px;
}
.report-invoice-preview :deep(p) {
  margin: 3px 0;
  white-space: pre-line;
  overflow-wrap: anywhere;
}
.report-invoice-preview :deep(.invoice-recipient) {
  margin: 10px 0;
}
.report-invoice-preview :deep(table) {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}
.report-invoice-preview :deep(th),
.report-invoice-preview :deep(td) {
  border: 1px solid #cbd5e1;
  padding: 4px;
  font-size: 9px;
  overflow-wrap: anywhere;
}
.report-invoice-preview :deep(footer) {
  margin-top: 14px;
}
@media (max-width: 900px) {
  .report-categories {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .document-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 600px) {
  .reports-mark {
    display: none;
  }
  .report-categories {
    gap: 10px;
  }
  .reports-hub .report-category {
    padding: 15px;
  }
  .report-category strong {
    font-size: 13px;
  }
  .report-category small {
    font-size: 10px;
  }
  .report-conclusion {
    padding: 20px;
  }
  .paper-letterhead {
    flex-direction: column;
  }
  .paper-letterhead > div:last-child {
    text-align: left;
  }
  .reports-dialog-overlay {
    padding: 8px;
  }
  .dialog-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }
  .dialog-scroll {
    padding: 12px;
  }
  .receipt-preview {
    padding: 18px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .reports-hub .report-category {
    transition: none;
  }
}
</style>
