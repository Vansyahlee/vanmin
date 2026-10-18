<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import FasSelect from '@/components/ui/FasSelect.vue'
import { decodeQRImage, parseCompanyQRIS } from '@/utils/qrisReader'
// Keep currency text exact even when cents exceed Number's safe integer range.
const money = value => { const [whole,decimal = ''] = String(value).split('.'); return 'Rp ' + whole.replace(/^0+(?=\d)/,'').replace(/\B(?=(\d{3})+(?!\d))/g,'.') + ',' + decimal.padEnd(2,'0') }
import { openFinancePrint, writeFinancePrint, downloadFinanceXlsx } from '@/utils/financeDocuments'
const emit = defineEmits(['record'])
const form = reactive({ channel:'BANK', kind:'SUPPLIER', provider:'', recipient:'', destination:'', amount:'', reference:'', description:'' })
const error = ref(''), notice = ref(''), review = ref(null), qr = ref(null), video = ref(null), camera = ref(false), scanning = ref(false), fileInput = ref(null)
let stream = null, timer = null, generation = 0
const amountLocked = computed(() => form.channel === 'QRIS' && Boolean(qr.value?.amount))
const stopCamera = () => { generation++; clearTimeout(timer); timer = null; stream?.getTracks().forEach(track=>track.stop()); stream = null; if (video.value) video.value.srcObject = null; camera.value = false; scanning.value = false }
const clearQR = () => { qr.value = null; form.recipient = ''; form.amount = ''; review.value = null }
const acceptQR = raw => { const parsed = parseCompanyQRIS(raw); qr.value = parsed; form.recipient = parsed.merchant; form.amount = parsed.amount; notice.value = 'QRIS terbaca. Nama merchant belum diverifikasi provider; dana belum dikirim.'; error.value = ''; stopCamera() }
const readFile = async event => {
 stopCamera(); error.value = ''; notice.value = ''; const file = event.target.files?.[0]; event.target.value = ''; if (!file) return
 clearQR()
 const ticket = generation
 if (!['image/png','image/jpeg','image/webp'].includes(file.type) || file.size > 10*1024*1024) { error.value = 'Gunakan JPG, PNG, atau WebP maksimal 10 MB.'; return }
 scanning.value = true; const url = URL.createObjectURL(file)
 try { const image = new Image(); image.src = url; await image.decode(); const raw = await decodeQRImage(image); if (ticket !== generation) return; if (!raw) throw new Error('QR tidak ditemukan. Gunakan gambar yang lebih jelas dan tidak terpotong.'); acceptQR(raw) }
 catch (failure) { if (ticket === generation) error.value = failure.message || 'Gambar tidak dapat dibaca.' }
 finally { URL.revokeObjectURL(url); if (ticket === generation) scanning.value = false }
}
const startCamera = async () => {
 stopCamera(); clearQR(); const ticket = generation; error.value = ''; notice.value = ''; scanning.value = true
 try {
  if (!navigator.mediaDevices?.getUserMedia) throw new Error('Kamera memerlukan HTTPS atau localhost. Gunakan unggah gambar jika kamera tidak tersedia.')
  const acquired = await navigator.mediaDevices.getUserMedia({video:{facingMode:{ideal:'environment'},width:{ideal:1280}},audio:false})
  if (ticket !== generation) { acquired.getTracks().forEach(track=>track.stop()); return }
  stream = acquired; camera.value = true; await nextTick(); if (ticket !== generation) return
  video.value.srcObject = stream; await video.value.play()
  const scan = async () => {
   if (ticket !== generation || !video.value) return
   try { const raw = await decodeQRImage(video.value); if (ticket !== generation) return; if (raw) { acceptQR(raw); scanning.value = false; return } }
   catch (failure) { if (ticket !== generation) return; error.value = failure.message; stopCamera(); scanning.value = false; return }
   timer = setTimeout(scan,400)
  }
  await scan()
 } catch (failure) { if (ticket === generation) { stopCamera(); error.value = failure.name === 'NotAllowedError' ? 'Izin kamera ditolak. Izinkan kamera atau unggah gambar QRIS.' : failure.message || 'Kamera tidak dapat dibuka.' } }
 finally { if (ticket === generation) scanning.value = false }
}
watch(() => form.channel, () => { stopCamera(); scanning.value = false; review.value = null; error.value = ''; notice.value = ''; qr.value = null; form.provider = ''; form.destination = ''; form.recipient = ''; form.amount = '' })
watch(form, () => { review.value = null }, {flush:'sync'})
const applyAmount = amount => { if (amountLocked.value) { error.value = 'Nominal QRIS ini ditentukan merchant. Kalkulator tidak dapat menggantinya.'; return }; form.amount = amount; notice.value = 'Hasil kalkulator diisikan ke nominal rencana pembayaran.' }
defineExpose({applyAmount})
const prepare = () => {
 error.value = ''; notice.value = ''; review.value = null
 const amount = form.amount.trim(), name = form.recipient.trim(), provider = form.provider.trim(), destination = form.destination.replace(/[\s-]/g,'')
 if (!name || !/^\d{1,15}(\.\d{1,2})?$/.test(amount) || Number(amount) <= 0 || !form.description.trim()) { error.value = 'Isi penerima, nominal positif tanpa pemisah ribuan, dan tujuan pembayaran.'; return }
 if (form.channel === 'BANK' && (!provider || !/^\d{6,30}$/.test(destination))) { error.value = 'Isi nama bank dan nomor rekening 6–30 digit. Nama penerima saja tidak cukup.'; return }
 if (form.channel === 'PHONE' && (!provider || !/^(?:08\d{8,11}|628\d{8,11}|\+628\d{8,11})$/.test(destination))) { error.value = 'Isi penyedia e-wallet dan nomor HP Indonesia yang terdaftar, contoh 081234567890.'; return }
 if (form.channel === 'QRIS' && !qr.value) { error.value = 'Unggah atau scan QRIS merchant terlebih dahulu.'; return }
 if (form.kind === 'SUPPLIER' && !form.reference.trim()) { error.value = 'Isi nomor invoice supplier untuk menghubungkan rencana pembayaran.'; return }
 if (form.kind === 'SALARY' && !form.reference.trim()) { error.value = 'Isi periode gaji, contoh Oktober 2026.'; return }
 review.value = { ...form, amount, recipient:name, provider, destination, reference:form.reference.trim(), description:form.description.trim(), city:qr.value?.city || '', createdAt:new Date().toLocaleString('id-ID'), status:'RENCANA — BELUM DIBAYAR' }
}
const rows = computed(() => review.value ? [
 ['Status',review.value.status],['Waktu dibuat',review.value.createdAt],['Jenis',({GENERAL:'Operasional',SALARY:'Gaji Karyawan',SUPPLIER:'Pembayaran Supplier'})[review.value.kind]],['Tujuan pembayaran',review.value.description],['Penerima (belum diverifikasi)',review.value.recipient],['Kanal',({BANK:'Rekening bank',PHONE:'Nomor HP / e-wallet',QRIS:'QRIS merchant'})[review.value.channel]],['Bank / e-wallet',review.value.provider || 'QRIS'],['Rekening / nomor HP',review.value.destination || 'Sesuai QRIS yang dibaca'],['Kota merchant',review.value.city || '—'],['Nominal',money(review.value.amount)],['Invoice / periode / referensi',review.value.reference || '—'],['Biaya provider','Belum tersedia'],['Keterangan','Instruksi internal. Bukan bukti transfer atau persetujuan pembayaran. Tidak mengubah Kas & Bank.'],
] : [])
const print = () => { try { writeFinancePrint(openFinancePrint(),'Rencana Pembayaran Perusahaan','BELUM DIBAYAR · Bukan bukti transfer',['Keterangan','Isi'],rows.value,{orientation:'portrait'}) } catch (failure) { error.value = failure.message } }
const record = () => { if (review.value?.channel === 'BANK') emit('record',{ ...review.value }) }
onBeforeUnmount(stopCamera)
</script>
<template>
 <section class="panel payment-workspace" aria-labelledby="payment-workspace-title">
  <header class="panel-heading"><div><p class="eyebrow">Pembayaran keluar</p><h2 id="payment-workspace-title">Siapkan pembayaran perusahaan</h2><small>Rekening supplier, gaji karyawan, dan QRIS merchant dalam satu tempat.</small></div><span class="badge">Transfer keluar belum terhubung</span></header>
  <div class="workspace-body">
   <p class="provider-note">Midtrans pembayaran customer sudah terpisah dari pengiriman dana perusahaan. Aktifkan layanan transfer keluar untuk validasi penerima dan kirim dana dari web. Saldo bank aktual belum tersedia.</p>
   <div class="channel-tabs" role="group" aria-label="Tujuan pembayaran"><button v-for="item in [{id:'BANK',label:'Rekening bank'},{id:'PHONE',label:'Nomor HP / e-wallet'},{id:'QRIS',label:'Scan QRIS'}]" :key="item.id" :class="{primary:form.channel === item.id}" :aria-pressed="form.channel === item.id" @click="form.channel = item.id">{{ item.label }}</button></div>
   <form @submit.prevent="prepare">
    <div class="form-grid">
     <label>Jenis pembayaran<FasSelect v-model="form.kind" aria-label="Jenis rencana pembayaran"><option value="GENERAL">Operasional</option><option value="SALARY">Gaji Karyawan</option><option value="SUPPLIER">Supplier</option></FasSelect></label>
     <label>{{ form.kind === 'SUPPLIER' ? 'Nomor invoice supplier *' : form.kind === 'SALARY' ? 'Periode gaji *' : 'Referensi (opsional)' }}<input v-model="form.reference" maxlength="120" :required="form.kind !== 'GENERAL'" :placeholder="form.kind === 'SALARY' ? 'Oktober 2026' : 'INV/SUP/001'" /></label>
     <template v-if="form.channel !== 'QRIS'"><label>{{ form.channel === 'BANK' ? 'Bank tujuan *' : 'Penyedia e-wallet *' }}<input v-model="form.provider" required maxlength="100" :placeholder="form.channel === 'BANK' ? 'BCA, Mandiri, BRI, atau bank lainnya' : 'GoPay, OVO, DANA, atau penyedia lainnya'" /></label><label>{{ form.channel === 'BANK' ? 'Nomor rekening *' : 'Nomor HP terdaftar *' }}<input v-model="form.destination" required type="text" inputmode="tel" autocomplete="off" maxlength="40" :placeholder="form.channel === 'BANK' ? 'Nomor rekening penerima' : '081234567890'" /></label></template>
     <div v-else class="span-two qr-box"><div class="actions"><button type="button" :disabled="scanning || camera" @click="fileInput.click()">Unggah gambar QRIS</button><button v-if="!camera" type="button" :disabled="scanning" @click="startCamera">Buka kamera</button><button v-else type="button" @click="stopCamera(); scanning = false">Matikan kamera</button></div><input ref="fileInput" type="file" accept="image/png,image/jpeg,image/webp" hidden @change="readFile" /><video v-if="camera" ref="video" autoplay muted playsinline aria-label="Pratinjau kamera QRIS" /><small>JPG, PNG, WebP · Maks. 10 MB · Gambar dibaca di perangkat ini. Foto dan kamera tidak diunggah.</small><p v-if="camera || scanning" role="status">{{ camera ? 'Arahkan kamera ke QRIS merchant…' : 'Membaca gambar…' }}</p><p v-if="qr" class="qr-merchant"><strong>{{ qr.merchant }}</strong> · {{ qr.city || 'Indonesia' }}<small>{{ qr.dynamic ? 'QRIS dinamis' : 'QRIS statis' }} · {{ qr.amount ? 'Nominal ditentukan merchant' : 'Isi nominal sesuai tagihan' }} · Belum dibayar</small></p></div>
     <label>{{ form.channel === 'QRIS' ? 'Nama merchant terbaca' : 'Nama penerima *' }}<input v-model="form.recipient" :aria-label="form.channel === 'QRIS' ? 'Nama merchant terbaca' : 'Nama penerima *'" required maxlength="150" :readonly="form.channel === 'QRIS'" placeholder="Nama sesuai rekening / e-wallet" /><small>Nama belum diverifikasi ke bank atau provider.</small></label>
     <label>Nominal (Rp) *<input v-model="form.amount" aria-label="Nominal (Rp) *" type="text" inputmode="decimal" required :readonly="amountLocked" maxlength="18" placeholder="2000000 atau 2000000.50" /><small>Tanpa titik ribuan. Gunakan kalkulator di sisi kanan.</small></label>
     <label class="span-two">Tujuan pembayaran *<input v-model="form.description" required maxlength="300" placeholder="Pembayaran invoice obat / gaji / operasional" /></label>
    </div>
    <p v-if="error" class="message error" role="alert">{{ error }}</p><p v-if="notice" class="message" role="status">{{ notice }}</p>
    <div class="actions workspace-actions"><button class="primary" type="submit" :disabled="camera || scanning">Periksa rencana pembayaran</button><small>Form ini belum mengirim dana dan belum mencatat pengeluaran.</small></div>
   </form>
   <section v-if="review" class="payment-review" aria-labelledby="payment-review-title"><h3 id="payment-review-title">Periksa tujuan & nominal</h3><span class="badge">{{ review.status }}</span><dl><div v-for="[label,value] in rows" :key="label"><dt>{{ label }}</dt><dd>{{ value }}</dd></div></dl><div class="actions"><button @click="print">Print / PDF Rencana</button><button @click="downloadFinanceXlsx('PTFAS_Rencana_Pembayaran.xlsx', [['RENCANA — BELUM DIBAYAR'],['Keterangan','Isi'],...rows])">Excel Rencana</button><button disabled>Kirim dana — belum tersedia</button></div><p v-if="review.channel === 'BANK'" class="muted">Jika pembayaran sudah dilakukan dan dikonfirmasi lewat bank, lanjutkan pencatatan dengan nomor referensi bank.</p><button v-if="review.channel === 'BANK'" @click="record">Sudah bayar lewat bank? Isi bukti pengeluaran</button><small>Rencana hanya ada selama halaman ini terbuka. Unduh jika perlu disimpan. Status lunas dan mutasi kas tidak dibuat dari rencana ini.</small></section>
  </div>
 </section>
</template>
<style scoped>
.workspace-body{padding:1.1rem}.provider-note{margin:0 0 1rem;padding:.8rem 1rem;background:var(--fas-overlay);border:1px solid var(--fas-border);border-radius:.8rem;color:var(--fas-muted);font-size:.78rem}.channel-tabs{display:grid;grid-template-columns:repeat(3,minmax(0,1fr));gap:.5rem;margin-bottom:1.2rem}.channel-tabs button{padding:.65rem!important}.workspace-actions{margin-top:1rem}.qr-box{padding:1rem;border:1px dashed var(--fas-accent-border);background:var(--fas-accent-soft);border-radius:.9rem}.qr-box video{display:block;max-width:100%;width:25rem;max-height:18rem;object-fit:contain;background:#101827;border-radius:.7rem;margin-top:.8rem}.qr-merchant{color:var(--fas-text)}.payment-review{margin-top:1.2rem;border-top:1px solid var(--fas-border);padding-top:1.2rem}.payment-review h3{font-size:1rem;font-weight:750;margin-bottom:.5rem}.payment-review dl{display:grid;grid-template-columns:1fr 1fr;gap:.85rem;margin:1rem 0}.payment-review dt{font-size:.7rem;color:var(--fas-muted)}.payment-review dd{margin:.25rem 0;color:var(--fas-text);overflow-wrap:anywhere;white-space:pre-wrap}@media(max-width:600px){.channel-tabs{grid-template-columns:1fr}.payment-review dl{grid-template-columns:1fr}}
</style>
