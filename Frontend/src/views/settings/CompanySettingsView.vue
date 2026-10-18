<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { companyProfile, loadCompanyProfile, saveCompanyProfile, getCompanyProfileHistory } from '@/services/company'
import { financeError, financeDate as date } from '@/composables/useFinanceLedger'
import '@/assets/finance-ledger.css'
const router = useRouter(), busy = ref(false), loaded = ref(false), error = ref(''), success = ref(''), history = ref([])
const form = reactive({ name: '', address: '', phone: '', email: '', tax_id: '', tax_nitku: '', license_number: '', maps_url: '', revision: 0 })
const refresh = async () => {
 busy.value = true; error.value = ''
 try { await loadCompanyProfile(); for (const key of Object.keys(form)) form[key] = companyProfile[key]; loaded.value = true; history.value = companyProfile.can_edit ? (await getCompanyProfileHistory()).data || [] : [] }
 catch (failure) { error.value = financeError(failure) } finally { busy.value = false }
}
const save = async () => {
 if (busy.value || !loaded.value || !companyProfile.can_edit) return
 busy.value = true; error.value = ''; success.value = ''
 try { await saveCompanyProfile({ ...form }); form.revision = companyProfile.revision; success.value = 'Profil perusahaan tersimpan dan dipakai bersama untuk dokumen baru.'; history.value = (await getCompanyProfileHistory()).data || [] }
 catch (failure) { error.value = financeError(failure) } finally { busy.value = false }
}
onMounted(refresh)
</script>
<template>
 <main class="finance-ledger"><header class="page-heading"><div><p class="eyebrow">Settings · Company</p><h1>Profil Perusahaan</h1><p class="muted">Satu sumber identitas dan alamat perusahaan untuk Finance, Distribution, faktur, serta dokumen cetak.</p></div><div class="actions"><button @click="router.push('/settings')">Kembali ke Settings</button><button :disabled="busy" @click="refresh">Muat Ulang Profil</button></div></header>
 <p v-if="error" class="message error" role="alert">{{ error }}</p><p v-if="success" class="message" role="status">{{ success }}</p>
 <p class="message">{{ companyProfile.can_edit ? 'Akun ini ditetapkan sebagai owner dan boleh mengubah profil.' : 'Mode lihat saja. Hanya akun owner yang ditetapkan di backend boleh menyimpan perubahan.' }} Alamat diketik manual; tautan Maps disimpan sebagai referensi, tidak mengambil alamat otomatis.</p>
 <form class="panel" @submit.prevent="save"><fieldset :disabled="busy || !loaded || !companyProfile.can_edit"><div class="filters">
 <label class="span-two">Nama perusahaan *<input v-model="form.name" required maxlength="200" /></label>
 <label>Telepon<input v-model="form.phone" maxlength="80" /></label><label>Email bisnis<input v-model="form.email" type="email" maxlength="254" /></label><label>NPWP<input v-model="form.tax_id" maxlength="80" /></label><label>NITKU (opsional)<input v-model="form.tax_nitku" maxlength="22" inputmode="numeric" /><small>22 digit yang terdaftar; dipakai sebagai isian awal draft pajak baru.</small></label><label>Nomor izin<input v-model="form.license_number" maxlength="100" /></label>
 <label class="span-two">Alamat manual perusahaan *<textarea v-model="form.address" required maxlength="1000" rows="4" /><small>Lengkapi nomor bangunan/RT/RW jika tersedia. Alamat awal mengikuti screenshot Maps yang dikirim.</small></label>
 <label class="span-two">Tautan lokasi Maps HTTPS<input v-model="form.maps_url" type="url" maxlength="2000" /></label>
 <div class="actions"><button class="primary" type="submit" :disabled="busy || !companyProfile.can_edit">{{ busy ? 'Memproses…' : 'Simpan Profil Perusahaan' }}</button></div></div></fieldset></form>
 <p class="muted">Faktur baru menyimpan identitas penerbit saat dibuat. Perpindahan alamat berikutnya tidak mengganti identitas pada faktur yang sudah memiliki snapshot. Faktur lama tanpa snapshot memakai profil pusat, lalu dibekukan saat dicetak melalui sistem.</p>
 <section v-if="companyProfile.can_edit" class="panel"><header class="panel-heading"><h2>50 Perubahan Profil Terakhir</h2></header><div class="table-scroll"><table><thead><tr><th>Waktu WIB</th><th>Owner</th><th>Nama / Alamat Sebelum</th><th>Nama / Alamat Sesudah</th></tr></thead><tbody><tr v-for="item in history" :key="item.id"><td>{{ date(item.changed_at) }}</td><td>{{ item.actor_name }}</td><td>{{ item.before_profile.name }}<small>{{ item.before_profile.address }}</small></td><td>{{ item.after_profile.name }}<small>{{ item.after_profile.address }}</small></td></tr><tr v-if="!history.length"><td colspan="4" class="empty">Belum ada perubahan profil.</td></tr></tbody></table></div></section>
 </main>
</template>
