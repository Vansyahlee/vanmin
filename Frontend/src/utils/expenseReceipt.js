import { companyDocumentHTML } from '@/services/company'

const escape = (value) =>
  String(value ?? '').replace(
    /[&<>"']/g,
    (char) =>
      ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[
        char
      ],
  )

// Internal expense receipt; payment execution stays outside this document.
export const writeExpenseReceipt = (popup, expense, rows) => {
  const title =
    expense.expense_kind === 'SALARY'
      ? 'Slip pembayaran gaji'
      : 'Struk pengeluaran'
  popup.document.open()
  popup.onload = () => {
    popup.focus()
    popup.print()
  }
  popup.document.write(
    `<!doctype html><html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>${escape(title)}</title><style>@page{size:80mm 297mm;margin:5mm}*{box-sizing:border-box}body{margin:20px auto;max-width:70mm;font:11px/1.6 Arial,sans-serif;color:#172033;background:white}h1{text-align:center;font-size:16px;margin:12px 0}h2{text-align:center;font-size:21px}p{overflow-wrap:anywhere}table{width:100%;border-collapse:collapse;border-top:1px dashed #64748b;border-bottom:1px dashed #64748b}td{vertical-align:top;padding:5px 0;overflow-wrap:anywhere}td:first-child{width:44%;padding-right:8px;color:#475569}td:last-child{text-align:right}tr{break-inside:avoid}.notice{font-size:10px;border-top:1px dashed #64748b;padding-top:10px}button{padding:10px;cursor:pointer}@media print{body{margin:0}button{display:none}}</style></head><body><button onclick="window.print()">Cetak / Simpan PDF</button>${companyDocumentHTML()}<h1>${escape(title)}</h1><p style="text-align:center">${escape(expense.expense_number)}<br>${escape(expense.expense_date)}</p><table><tbody>${rows.map(([label, value]) => `<tr><td>${escape(label)}</td><td>${escape(value)}</td></tr>`).join('')}</tbody></table><p class="notice">BUKTI PENCATATAN INTERNAL — bukan bukti transfer bank. ${expense.status === 'VOID' ? 'CATATAN DIBATALKAN. Mutasi pembalik dicatat terpisah.' : 'Pembayaran dicatat oleh petugas.'} Saldo sebelum/sesudah merupakan saldo buku per akun menurut tanggal dan urutan mutasi. Pencatatan mundur tanggal dapat mengubah saldo historis. Tidak menunjukkan saldo bank aktual.</p></body></html>`,
  )
  popup.document.close()
}
