import { reactive } from 'vue'
import api from '@/services/api'
export const companyProfile = reactive({ name: '', address: '', phone: '', email: '', tax_id: '', tax_nitku: '', license_number: '', maps_url: '', revision: 0, loaded: false, can_edit: false })
let pending
export const loadCompanyProfile = async () => {
 if (pending) return pending
 const token = localStorage.getItem('fas_token')
 pending = api.get('/company-profile').then(response => {
  if (localStorage.getItem('fas_token') !== token) throw new Error('Sesi berubah. Muat ulang profil perusahaan.')
  Object.assign(companyProfile, response.data.data, { loaded: true, can_edit: Boolean(response.data.can_edit) })
  return response.data
 }).finally(() => { pending = null })
 return pending
}
export const saveCompanyProfile = async payload => {
 const response = await api.put('/company-profile', payload)
 Object.assign(companyProfile, response.data.data, { loaded: true, can_edit: Boolean(response.data.can_edit) })
 window.dispatchEvent(new Event('fas-company-profile-changed'))
 return response.data
}
export const getCompanyProfileHistory = async () => (await api.get('/company-profile/history')).data
const escape = value => String(value || '').replace(/[&<>"']/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[c])
export const companyDocumentRows = () => {
 if (!companyProfile.loaded) throw new Error('Profil perusahaan belum termuat. Muat ulang halaman atau periksa migrasi profil perusahaan.')
 return [[companyProfile.name], [companyProfile.address], [`Telepon: ${companyProfile.phone || '—'} · Email: ${companyProfile.email || '—'}`], [`NPWP: ${companyProfile.tax_id || '—'} · Izin: ${companyProfile.license_number || '—'}`]]
}
export const companyDocumentHTML = () => `<div style="margin-bottom:16px;white-space:pre-line;font-size:11px">${companyDocumentRows().map(row => `<div>${escape(row[0])}</div>`).join('')}</div>`
