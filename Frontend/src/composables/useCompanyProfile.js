import { onBeforeUnmount, onMounted, watch } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { companyProfile, loadCompanyProfile } from '@/services/company'
export const useCompanyProfile = () => {
 const auth = useAuthStore()
 const refresh = () => { if (auth.user && document.visibilityState !== 'hidden') loadCompanyProfile().catch(() => {}) }
 let timer
 watch(() => auth.user?.id, id => { companyProfile.can_edit = false; if (id) refresh(); else companyProfile.loaded = false }, { immediate: true })
 onMounted(() => { window.addEventListener('focus', refresh); window.addEventListener('fas-company-profile-changed', refresh); timer = setInterval(refresh, 60000) })
 onBeforeUnmount(() => { window.removeEventListener('focus', refresh); window.removeEventListener('fas-company-profile-changed', refresh); clearInterval(timer) })
 return { companyProfile, loadCompanyProfile }
}
