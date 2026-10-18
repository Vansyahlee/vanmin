import { companyDocumentHTML, companyDocumentRows } from '@/services/company'
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

export const buildFinanceXlsx = rows => {
  rows = [...companyDocumentRows(), [], ...rows]
  const cells = rows.map((row, r) => `<row r="${r + 1}">${row.map((value, c) => typeof value === 'number' && Number.isFinite(value) ? `<c r="${column(c)}${r + 1}"><v>${value}</v></c>` : `<c r="${column(c)}${r + 1}" t="inlineStr"><is><t xml:space="preserve">${xml(value)}</t></is></c>`).join('')}</row>`).join('')
  return zip([
    ['[Content_Types].xml', '<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>'],
    ['_rels/.rels', '<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>'],
    ['xl/workbook.xml', '<?xml version="1.0"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="Laporan" sheetId="1" r:id="rId1"/></sheets></workbook>'],
    ['xl/_rels/workbook.xml.rels', '<?xml version="1.0"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>'],
    ['xl/worksheets/sheet1.xml', `<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>${cells}</sheetData></worksheet>`],
  ])
}

export const downloadFinanceXlsx = (filename, rows) => {
  const blob = buildFinanceXlsx(rows)
  const url = URL.createObjectURL(blob), link = document.createElement('a')
  link.href = url; link.download = filename; document.body.appendChild(link); link.click(); link.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

export const openFinancePrint = () => {
  const popup = window.open('', '_blank')
  if (!popup) throw new Error('Izinkan pop-up browser untuk membuka dokumen cetak.')
  popup.opener = null
  popup.document.write('<p>Menyiapkan dokumen…</p>')
  return popup
}

export const writeFinancePrint = (popup, title, description, headings, rows, options = {}) => {
  const orientation = options.orientation === 'portrait' ? 'portrait' : 'landscape'
  popup.document.open()
  popup.onload = () => { popup.focus(); popup.print() }
  popup.document.write(`<!doctype html><html lang="id"><head><meta charset="UTF-8"><title>${xml(title)}</title><style>@page{size:A4 ${orientation};margin:12mm}body{font:11px Arial,sans-serif;color:#172033}h1{font-size:20px}p{line-height:1.6}table{border-collapse:collapse;width:100%}th,td{padding:7px;border:1px solid #cbd5e1;text-align:left;overflow-wrap:anywhere}th{background:#f1f5f9}thead{display:table-header-group}tr{break-inside:avoid}@media print{button{display:none}}</style></head><body>${companyDocumentHTML()}<button onclick="window.print()">Cetak / Simpan PDF</button><h1>${xml(title)}</h1><p>${xml(description)}</p><table><thead><tr>${headings.map(h => `<th>${xml(h)}</th>`).join('')}</tr></thead><tbody>${rows.map(row => `<tr>${row.map(cell => `<td>${xml(cell)}</td>`).join('')}</tr>`).join('')}</tbody></table></body></html>`)
  popup.document.close()
}
