const MAX_CENTS = 9999999999999999n
const cents = raw => {
 const text = String(raw ?? '').trim() || '0'
 if (!/^\d{1,14}(\.\d{1,2})?$/.test(text)) return null
 const [whole, fraction = ''] = text.split('.')
 const value = BigInt(whole) * 100n + BigInt(fraction.padEnd(2, '0'))
 return value <= MAX_CENTS ? value : null
}
export const salaryNetAmount = (basic, allowance, deduction) => {
 const values = [basic, allowance, deduction].map(cents)
 if (values.some(value => value === null)) return ''
 const [b, a, d] = values, gross = b + a, net = gross - d
 return gross <= MAX_CENTS && net > 0n ? `${net / 100n}.${String(net % 100n).padStart(2, '0')}` : ''
}
export const expenseKindLabel = kind => ({ GENERAL: 'Operasional', SALARY: 'Gaji Karyawan', SUPPLIER: 'Pembayaran Supplier' })[kind] || 'Operasional'
export const expenseDetailsRows = (item, money) => {
 const d = item.details || {}
 if (item.expense_kind === 'SALARY') return [['Periode gaji', d.payroll_period], ['Kode karyawan', d.employee_code || '—'], ['Gaji pokok', money(d.basic_salary)], ['Tunjangan', money(d.allowance)], ['Potongan', money(d.deduction)], ['Gaji bersih dibayar', money(item.amount)]]
 if (item.expense_kind === 'SUPPLIER') return [['Nomor invoice supplier', d.supplier_invoice_number], ['Tanggal invoice supplier', d.supplier_invoice_date || '—'], ['Referensi PO / pembelian', d.purchase_reference || '—']]
 return []
}
