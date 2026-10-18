import { reportCents, reportDecimal } from '@/utils/financeReporting'
const clean=value=>String(value??'').replace(/[\u0000-\u0008\u000b\u000c\u000e-\u001f]/g,'')
const xml=value=>clean(value).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&apos;'}[c]))
export const digits=value=>String(value||'').replace(/\D/g,'')
export const makeCoretaxDraft=(source,company)=>({
 seller_tin:digits(source.issuer_profile?.tax_id || company.tax_id),seller_nitku:source.issuer_profile?.tax_nitku || company.tax_nitku||'',buyer_tin:digits(source.buyer_tin),buyer_nitku:'',buyer_name:source.buyer_name||source.store_name,buyer_address:source.buyer_address||'',buyer_email:'',tax_date:source.invoice_date,trx_code:'',pkp_confirmed:false,notes:'',
 items:source.items.map(item=>({source_id:item.id,name:item.name,qty:item.qty,price:item.price,discount:item.discount,base:item.subtotal,other_base:'',vat_rate:source.tax_rate,vat:'',code:'',unit:''}))
})
const round=(n,d)=>(n*2n+d)/(2n*d)
export const validateCoretaxDraft=(doc,source)=>{
 const errors=[]
 if(!doc||!source)return ['Faktur belum tersedia.']
 if(source.tax_mode==='NON_PPN'||reportCents(source.vat)<=0n)errors.push('Ekspor ini hanya mendukung faktur PPN domestik normal. Faktur non-PPN/PPN nol tidak dikonversi otomatis.')
 if(!doc.pkp_confirmed)errors.push('Konfirmasi status PKP dan pemeriksaan data oleh Finance.')
 for(const [key,label] of [['seller_tin','NPWP penjual'],['buyer_tin','NPWP pembeli']])if(!/^\d{16}$/.test(doc[key]||''))errors.push(label+' wajib 16 digit terdaftar.')
 for(const [key,tin,label] of [['seller_nitku','seller_tin','NITKU penjual'],['buyer_nitku','buyer_tin','NITKU pembeli']])if(!/^\d{22}$/.test(doc[key]||'')||!doc[key].startsWith(doc[tin]))errors.push(label+' wajib 22 digit dan sesuai NPWP. Jangan membuat NITKU sendiri.')
 if(digits(source.buyer_tin).length===16&&doc.buyer_tin!==digits(source.buyer_tin))errors.push('NPWP pembeli berbeda dari faktur sumber. Periksa data customer terlebih dahulu.')
 if(!doc.buyer_name?.trim()||!doc.buyer_address?.trim())errors.push('Lengkapi nama dan alamat perpajakan pembeli.')
 if(doc.buyer_email&&!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(doc.buyer_email))errors.push('Email pembeli tidak valid.')
 if(!['01','04'].includes(doc.trx_code))errors.push('Pilih kode transaksi 01 atau 04 sesuai perlakuan pajak. Jenis lain belum didukung ekspor ini.')
 if(!/^\d{4}-\d{2}-\d{2}$/.test(doc.tax_date||'')||Number.isNaN(Date.parse(doc.tax_date))||new Date(doc.tax_date).toISOString().slice(0,10)!==doc.tax_date)errors.push('Tanggal faktur pajak tidak valid.')
 if(!Array.isArray(doc.items)||doc.items.length!==source.items.length||!doc.items.length)return [...errors,'Rincian tidak sama dengan faktur sumber. Muat ulang draft.']
 let baseTotal=0n,vatTotal=0n,otherTotal=0n;const seen=new Set()
 for(const [index,item] of doc.items.entries()){
  const label='Baris '+(index+1)+': ',original=source.items.find(x=>x.id===item.source_id)
  if(!original||seen.has(item.source_id)||String(item.qty)!==String(original.qty)){errors.push(label+'item/qty berbeda dari faktur sumber.');continue};seen.add(item.source_id)
  if(!/^\d{6}$/.test(item.code||'')||!/^UM\.\d{4}$/.test(item.unit||'')||!item.name?.trim())errors.push(label+'lengkapi kode barang 6 digit, nama, dan kode satuan DJP UM.xxxx.')
  try {
   const price=reportCents(item.price),discount=reportCents(item.discount),base=reportCents(item.base),other=reportCents(item.other_base),rate=reportCents(item.vat_rate),vat=reportCents(item.vat)
   if(!/^\d{1,10}$/.test(item.qty)||BigInt(item.qty)<=0n||[price,discount,base,other,rate,vat].some(x=>x<0n)||rate>10000n)throw new Error()
   if(price*BigInt(item.qty)-discount!==base)errors.push(label+'harga × qty − diskon harus sama dengan TaxBase (harga sebelum PPN).')
   if(doc.trx_code==='01'&&other!==base)errors.push(label+'untuk kode 01, OtherTaxBase harus sama dengan TaxBase pada subset ekspor ini.')
   if(rate!==reportCents(source.tax_rate))errors.push(label+'tarif berbeda dari snapshot tarif faktur sumber.')
   const difference=vat-round(other*rate,10000n)
   if(difference>1n||difference< -1n)errors.push(label+'PPN tidak sesuai dasar pengenaan dan tarif (toleransi 1 sen).')
   baseTotal+=base;vatTotal+=vat;otherTotal+=other
  }catch{errors.push(label+'nominal harus nonnegatif, maksimal 2 desimal dan qty bilangan bulat positif.')}
 }
 if(otherTotal!==reportCents(source.tax_base))errors.push('Total OtherTaxBase/DPP PPN harus sama dengan DPP PPN faktur sumber.')
 if(vatTotal!==reportCents(source.vat))errors.push('Total PPN rincian harus sama dengan PPN faktur sumber.')
 if(baseTotal+vatTotal!==reportCents(source.total))errors.push('Total harga sebelum PPN + PPN harus sama dengan total faktur sumber. Periksa diskon faktur dan harga termasuk PPN.')
 return errors
}
export const buildCoretaxXML=(doc,source)=>{
 const errors=validateCoretaxDraft(doc,source);if(errors.length)throw new Error(errors.join('\n'))
 const tag=(name,value)=>`<${name}>${xml(value)}</${name}>`,money=value=>reportDecimal(reportCents(value))
 // Field names/order mirror DJP Sample Faktur PK Template v1.4, 1 March 2025.
 return '<?xml version="1.0" encoding="utf-8"?>\n<TaxInvoiceBulk xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="TaxInvoice.xsd">'+tag('TIN',doc.seller_tin)+'<ListOfTaxInvoice><TaxInvoice>'+tag('TaxInvoiceDate',doc.tax_date)+tag('TaxInvoiceOpt','Normal')+tag('TrxCode',doc.trx_code)+'<AddInfo/><CustomDoc/><CustomDocMonthYear/>'+tag('RefDesc',source.invoice_number)+'<FacilityStamp/>'+tag('SellerIDTKU',doc.seller_nitku)+tag('BuyerTin',doc.buyer_tin)+tag('BuyerDocument','TIN')+tag('BuyerCountry','IND')+'<BuyerDocumentNumber/>'+tag('BuyerName',doc.buyer_name)+tag('BuyerAdress',doc.buyer_address)+tag('BuyerEmail',doc.buyer_email)+tag('BuyerIDTKU',doc.buyer_nitku)+'<ListOfGoodService>'+doc.items.map(item=>'<GoodService>'+tag('Opt','A')+tag('Code',item.code)+tag('Name',item.name)+tag('Unit',item.unit)+tag('Price',money(item.price))+tag('Qty',item.qty)+tag('TotalDiscount',money(item.discount))+tag('TaxBase',money(item.base))+tag('OtherTaxBase',money(item.other_base))+tag('VATRate',money(item.vat_rate))+tag('VAT',money(item.vat))+'<STLGRate>0</STLGRate><STLG>0</STLG></GoodService>').join('')+'</ListOfGoodService></TaxInvoice></ListOfTaxInvoice></TaxInvoiceBulk>'
}
export const downloadCoretaxXML=(doc,source)=>{
 const text=buildCoretaxXML(doc,source),url=URL.createObjectURL(new Blob([text],{type:'application/xml;charset=utf-8'})),a=document.createElement('a')
 a.href=url;a.download='Draft_Coretax_'+source.invoice_number.replace(/[^A-Za-z0-9_-]/g,'_')+'.xml';document.body.appendChild(a);a.click();a.remove();setTimeout(()=>URL.revokeObjectURL(url),1000)
}
