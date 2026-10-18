export const qrisCRC = text => {
 let crc = 0xffff
 for (const byte of new TextEncoder().encode(text)) { crc ^= byte << 8; for (let bit = 0; bit < 8; bit++) crc = (crc & 0x8000 ? (crc << 1) ^ 0x1021 : crc << 1) & 0xffff }
 return crc.toString(16).toUpperCase().padStart(4,'0')
}
export const parseCompanyQRIS = raw => {
 const payload = String(raw).trim()
 if (payload.length > 4096 || !/^[\x20-\x7e]+$/.test(payload)) throw new Error('QR bukan payload QRIS yang didukung.')
 const fields = {}, order = []; let offset = 0
 while (offset < payload.length) {
  const head = payload.slice(offset,offset+4); if (!/^\d{4}$/.test(head)) throw new Error('Struktur QR tidak valid.')
  const id = head.slice(0,2), length = Number(head.slice(2)); offset += 4
  if (!length || offset+length > payload.length || Object.hasOwn(fields,id)) throw new Error('Panjang atau tag QR tidak valid.')
  fields[id] = payload.slice(offset,offset+length); order.push(id); offset += length
 }
 if (fields['00'] !== '01' || fields['53'] !== '360' || fields['58'] !== 'ID' || !fields['59'] || !Object.entries(fields).some(([id,v]) => Number(id) >= 26 && Number(id) <= 51 && /ID\.CO\./i.test(v))) throw new Error('QR ini bukan QRIS Rupiah Indonesia yang dikenali.')
 if (order.at(-1) !== '63' || fields['63']?.length !== 4 || !/^[A-Fa-f\d]{4}$/.test(fields['63']) || qrisCRC(payload.slice(0,-4)) !== fields['63'].toUpperCase()) throw new Error('Checksum QRIS tidak cocok. Gunakan gambar asli yang jelas.')
 if (fields['01'] && !['11','12'].includes(fields['01'])) throw new Error('Jenis QRIS belum didukung.')
 if (fields['54'] && (!/^\d+(\.\d{1,2})?$/.test(fields['54']) || Number(fields['54']) <= 0 || fields['54'].split('.')[0].length > 15)) throw new Error('Nominal QRIS tidak valid.')
 return { payload, merchant: fields['59'], city: fields['60'] || '', amount: fields['54'] || '', dynamic: fields['01'] === '12' }
}
export const decodeQRImage = async image => {
 const width = image.videoWidth || image.naturalWidth || image.width, height = image.videoHeight || image.naturalHeight || image.height
 if (!width || !height) return null
 const scale = Math.min(1,1600/Math.max(width,height)), canvas = document.createElement('canvas')
 canvas.width = Math.round(width*scale); canvas.height = Math.round(height*scale)
 const context = canvas.getContext('2d',{willReadFrequently:true}); context.drawImage(image,0,0,canvas.width,canvas.height)
 const pixels = context.getImageData(0,0,canvas.width,canvas.height)
 const { default: jsQR } = await import('@/vendor/jsQR')
 return jsQR(pixels.data,pixels.width,pixels.height,{ inversionAttempts:'attemptBoth' })?.data || null
}
