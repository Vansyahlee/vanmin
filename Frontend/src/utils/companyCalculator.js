// Exact rational arithmetic. No eval/Function and no binary floating-point money.
export const calculateCompanyExpression = expression => {
 const source = String(expression).replace(/,/g, '.').replace(/×/g, '*').replace(/÷/g, '/').replace(/−/g, '-').replace(/\s/g, '')
 if (!source || source.length > 160 || /[^\d.+*/%()-]/.test(source)) throw new Error('Isi perhitungan dengan angka dan operator yang tersedia.')
 let position = 0
 const gcd = (a,b) => { a = a < 0n ? -a : a; while (b) [a,b] = [b,a % b]; return a || 1n }
 const value = (n,d = 1n) => { if (!d) throw new Error('Tidak dapat membagi dengan nol.'); if (d < 0n) { n = -n; d = -d }; const g = gcd(n,d); n /= g; d /= g; if (n.toString().length > 150 || d.toString().length > 150) throw new Error('Perhitungan terlalu besar.'); return { n,d } }
 const operation = (a,b,op) => op === '+' ? value(a.n*b.d+b.n*a.d,a.d*b.d) : op === '-' ? value(a.n*b.d-b.n*a.d,a.d*b.d) : op === '*' ? value(a.n*b.n,a.d*b.d) : value(a.n*b.d,a.d*b.n)
 const atom = () => {
  let result
  if (source[position] === '+' || source[position] === '-') { const sign = source[position++]; result = atom(); if (sign === '-') result = value(-result.n,result.d); return result }
  if (source[position] === '(') { position++; result = sum(); if (source[position++] !== ')') throw new Error('Tutup tanda kurung terlebih dahulu.') }
  else { const match = source.slice(position).match(/^(?:\d+(?:\.\d*)?|\.\d+)/); if (!match || match[0].length > 24) throw new Error('Susunan angka belum lengkap.'); position += match[0].length; const [whole,fraction = ''] = match[0].split('.'); result = value(BigInt((whole || '0') + fraction),10n ** BigInt(fraction.length)) }
  while (source[position] === '%') { position++; result = value(result.n,result.d*100n) }
  return result
 }
 const product = () => { let result = atom(); while (source[position] === '*' || source[position] === '/') { const op = source[position++]; result = operation(result,atom(),op) }; return result }
 const sum = () => { let result = product(); while (source[position] === '+' || source[position] === '-') { const op = source[position++]; result = operation(result,product(),op) }; return result }
 const result = sum(); if (position !== source.length) throw new Error('Periksa kembali susunan perhitungan.')
 const absolute = result.n < 0n ? -result.n : result.n
 const cents = (absolute * 100n * 2n + result.d) / (result.d * 2n)
 return `${result.n < 0n && cents ? '-' : ''}${cents / 100n}.${String(cents % 100n).padStart(2,'0')}`
}
