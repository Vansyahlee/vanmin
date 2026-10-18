// Monetary values stay decimal strings. Unknown input must never become zero profit.
export const reportCents = value => {
 const text = String(value ?? '')
 if (!/^-?\d{1,36}(\.\d{1,2})?$/.test(text)) throw new Error('Nominal harus angka maksimal 2 desimal, tanpa pemisah ribuan.')
 const negative = text.startsWith('-'), [whole,fraction=''] = text.replace('-','').split('.')
 const n = BigInt(whole)*100n+BigInt(fraction.padEnd(2,'0'))
 return negative ? -n : n
}
export const reportDecimal = value => { const n = value < 0n ? -value : value; return `${value < 0n ? '-' : ''}${n/100n}.${String(n%100n).padStart(2,'0')}` }
export const reportMoney = value => {
 if (value === null || value === undefined || value === '') return 'Belum lengkap'
 const text = reportDecimal(reportCents(value)),[whole,fraction] = text.split('.')
 return `Rp ${whole.replace(/\B(?=(\d{3})+(?!\d))/g,'.')},${fraction}`
}
const roundDivide = (n,d) => { if (!d) return 0n; const sign = n < 0n ? -1n : 1n; const a = n < 0n ? -n : n; return sign*((a*2n+d)/(2n*d)) }
export const reportPercent = (amount,total) => {
 const denominator=reportCents(total); if (denominator <= 0n) return '—'
 return reportDecimal(roundDivide(reportCents(amount)*10000n,denominator)).replace('.',',')+'%'
}
export const periodProfit = row => {
 const s=row?.settings
 if (!s || row.full_month === false || ['hpp','salary_expense','operating_expense','income_tax','other_income','owner_percent'].some(key => !/^\d+(\.\d{1,2})?$/.test(String(s[key] ?? '')))) return null
 const revenue=reportCents(row.revenue), hpp=reportCents(s.hpp), salary=reportCents(s.salary_expense),opex=reportCents(s.operating_expense),tax=reportCents(s.income_tax),other=reportCents(s.other_income),percent=reportCents(s.owner_percent)
 if(percent>10000n)return null
 const gross=revenue-hpp,operating=gross-salary-opex,pretax=operating+other,net=pretax-tax,owner=net>0n?roundDivide(net*percent,10000n):0n
 return Object.fromEntries(Object.entries({gross,operating,pretax,net,owner,retained:net-owner}).map(([key,value])=>[key,reportDecimal(value)]))
}
export const monthDates = month => {
 if (!/^\d{4}-(0[1-9]|1[0-2])$/.test(month)) throw new Error('Pilih bulan yang valid.')
 const [year,m] = month.split('-').map(Number),last=new Date(Date.UTC(year,m,0)).getUTCDate()
 return {date_from:month+'-01',date_to:month+'-'+last}
}
export const profitRows = row => {
 const p=periodProfit(row),s=row?.settings || {},revenue=row?.revenue || '0'
 return [ ['Penjualan sebelum PPN',revenue],['HPP barang terjual',s.hpp],['Laba kotor',p?.gross],['Beban gaji periode',s.salary_expense],['Beban operasional periode',s.operating_expense],['Laba usaha',p?.operating],['Pendapatan lain',s.other_income],['Laba sebelum pajak',p?.pretax],['Beban pajak penghasilan',s.income_tax],['Laba bersih manajerial',p?.net],['Simulasi bagian owner',p?.owner],['Sisa laba setelah simulasi',p?.retained] ].map(([label,amount])=>[label,reportMoney(amount),amount == null || amount === '' ? '—' : reportPercent(amount,revenue)])
}
