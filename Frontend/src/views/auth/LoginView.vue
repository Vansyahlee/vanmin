<script setup>
import PharmaceuticalScene from '@/components/brand/PharmaceuticalScene.vue'
import { ref } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const errorMessage = ref('')

const login = async () => {
  errorMessage.value = ''
  loading.value = true

  try {
    const loggedInUser = await authStore.login(email.value, password.value)
    const landingPath = loggedInUser?.role === 'pegawai'
      ? '/warehouse'
      : loggedInUser?.role === 'finance'
        ? '/finance'
        : '/dashboard'
    await router.replace(landingPath)
  } catch (error) {
    errorMessage.value = error.response?.data?.error || 'Gagal masuk ke sistem'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <main class="fas-login">
    <div class="login-shell">
      <section class="login-brand" aria-label="PT FAS Business Platform">
        <div class="brand-heading">
          <span class="brand-mark">F</span>
          <div>
            <strong>PT. FAS</strong>
            <p>FURI ANUGERAH SEJAHTERA</p>
          </div>
        </div>
        <div class="brand-story">
          <p class="brand-eyebrow">CONNECTED CARE. CONNECTED BUSINESS.</p>
          <h1>Setiap koneksi, langkah lebih berarti.</h1>
          <p class="brand-description">
            Dari pesanan hingga laporan, bantu setiap bagian bekerja dengan data
            yang saling terhubung.
          </p>
          <PharmaceuticalScene compact />
        </div>
        <p class="brand-footer">PT. FAS Business Operations Platform</p>
      </section>
      <section class="login-form-panel">
        <div class="login-form-wrap">
          <RouterLink to="/" class="login-back"
            >← Halaman depan PT FAS</RouterLink
          >
          <p class="login-eyebrow">Akses internal perusahaan</p>
          <h2>Selamat datang kembali.</h2>
          <p class="login-description">
            Masuk dengan akun PT. FAS untuk melanjutkan pekerjaan.
          </p>
          <form @submit.prevent="login">
            <label for="email"
              >Email<input
                id="email"
                v-model="email"
                type="email"
                required
                autocomplete="username"
                placeholder="nama@perusahaan.com"
                :disabled="loading"
            /></label>
            <label for="password"
              >Password<span class="password-field"
                ><input
                  id="password"
                  v-model="password"
                  :type="showPassword ? 'text' : 'password'"
                  required
                  autocomplete="current-password"
                  placeholder="Masukkan password"
                  :disabled="loading" /><button
                  type="button"
                  class="password-toggle"
                  :aria-label="
                    showPassword ? 'Sembunyikan password' : 'Tampilkan password'
                  "
                  :aria-pressed="showPassword"
                  @click="showPassword = !showPassword"
                >
                  <svg
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    stroke-width="1.8"
                    aria-hidden="true"
                  >
                    <path
                      d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12Z"
                    />
                    <circle cx="12" cy="12" r="3" />
                    <path v-if="showPassword" d="m3 3 18 18" />
                  </svg></button></span
            ></label>
            <p v-if="errorMessage" class="login-error" role="alert">
              {{ errorMessage }}
            </p>
            <button
              class="login-submit"
              type="submit"
              :disabled="loading"
              :aria-busy="loading"
            >
              {{ loading ? 'Memproses…' : 'Masuk ke sistem'
              }}<span aria-hidden="true">→</span>
            </button>
          </form>
          <p class="login-footnote">
            Gunakan akun yang sudah didaftarkan oleh administrator perusahaan.
          </p>
        </div>
      </section>
    </div>
  </main>
</template>

<style scoped>
.fas-login {
  min-height: 100dvh;
  background: var(--fas-bg);
  color: var(--fas-text);
  padding: clamp(1rem, 3vw, 3rem);
  display: grid;
  align-items: center;
}
.login-shell {
  width: min(1180px, 100%);
  margin: auto;
  display: grid;
  grid-template-columns: 1.05fr 1fr;
  border: 1px solid var(--fas-border);
  border-radius: 1.75rem;
  overflow: hidden;
  background: var(--fas-card);
  box-shadow: var(--fas-shadow-card);
}
.login-brand {
  padding: clamp(1.5rem, 4vw, 3.5rem);
  background: linear-gradient(
    145deg,
    var(--fas-accent-soft),
    var(--fas-elevated)
  );
  border-right: 1px solid var(--fas-border);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 3rem;
}
.brand-heading {
  display: flex;
  align-items: center;
  gap: 0.9rem;
}
.brand-mark {
  width: 3rem;
  height: 3rem;
  display: grid;
  place-items: center;
  border: 1px solid var(--fas-accent-border);
  border-radius: 1rem;
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
  font-size: 1.5rem;
  font-weight: 800;
}
.brand-heading strong {
  font-size: 1.05rem;
}
.brand-heading p {
  font-size: 0.65rem;
  letter-spacing: 0.12em;
  margin: 0.25rem 0 0;
  color: var(--fas-muted);
}
.brand-eyebrow,
.login-eyebrow {
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--fas-accent-text);
  letter-spacing: 0.06em;
}
.brand-story h1 {
  font-size: clamp(1.8rem, 2.8vw, 2.65rem);
  font-weight: 750;
  line-height: 1.2;
  letter-spacing: -0.04em;
  margin: 0.9rem 0 1rem;
  max-width: 14ch;
}
.brand-description {
  color: var(--fas-text-soft);
  font-size: 0.9rem;
  line-height: 1.8;
}
.brand-modules {
  display: grid;
  gap: 0.8rem;
  margin-top: 1.8rem;
}
.brand-modules article {
  display: flex;
  align-items: center;
  gap: 0.85rem;
  border: 1px solid var(--fas-border);
  border-radius: 1rem;
  padding: 0.85rem 1rem;
  background: var(--fas-card);
}
.brand-modules article > span {
  font-size: 0.75rem;
  color: var(--fas-accent-text);
  font-weight: 750;
}
.brand-modules h2 {
  font-size: 0.85rem;
  font-weight: 700;
  margin: 0;
}
.brand-modules p {
  font-size: 0.75rem;
  color: var(--fas-muted);
  margin: 0.25rem 0 0;
  line-height: 1.5;
}
.brand-footer {
  font-size: 0.7rem;
  color: var(--fas-muted);
  margin: 0;
}
.login-form-panel {
  padding: clamp(1.5rem, 4vw, 3.5rem);
  display: grid;
  align-items: center;
  background: var(--fas-card);
}
.login-form-wrap {
  width: 100%;
  max-width: 400px;
  margin: auto;
}
.login-back {
  display: inline-block;
  font-size: 0.8rem;
  color: var(--fas-muted);
  margin-bottom: 3rem;
}
.login-back:hover {
  color: var(--fas-accent-text);
}
.login-form-wrap h2 {
  font-size: 2rem;
  line-height: 1.25;
  font-weight: 750;
  letter-spacing: -0.035em;
  margin: 0.6rem 0;
}
.login-description {
  color: var(--fas-muted);
  font-size: 0.88rem;
  line-height: 1.7;
  margin-bottom: 1.8rem;
}
.login-form-wrap form {
  display: grid;
  gap: 1.2rem;
}
.login-form-wrap label {
  display: grid;
  gap: 0.55rem;
  font-size: 0.82rem;
  font-weight: 650;
  color: var(--fas-text-soft);
}
.login-form-wrap input {
  display: block;
  width: 100%;
  min-width: 0;
  min-height: 3.1rem;
  padding: 0.8rem 1rem;
  border: 1px solid var(--fas-border-strong);
  border-radius: 0.85rem;
  background: var(--fas-input);
  color: var(--fas-text);
  font: inherit;
  font-weight: 400;
  outline: none;
}
.login-form-wrap input::placeholder {
  color: var(--fas-muted);
}
.password-field {
  display: block;
  position: relative;
}
.password-field input::-ms-reveal,
.password-field input::-ms-clear {
  display: none;
}
.password-field input {
  padding-right: 3.4rem;
}
.password-toggle {
  position: absolute;
  right: 0.4rem;
  top: 0.35rem;
  display: grid;
  place-items: center;
  width: 2.4rem;
  height: 2.4rem;
  border: 0;
  border-radius: 0.6rem;
  background: transparent;
  color: var(--fas-muted);
  cursor: pointer;
}
.password-toggle svg {
  width: 1.2rem;
  height: 1.2rem;
}
.password-toggle:hover {
  background: var(--fas-accent-soft);
  color: var(--fas-accent-text);
}
.login-submit {
  display: flex;
  justify-content: space-between;
  align-items: center;
  min-height: 3.1rem;
  border: 1px solid var(--fas-accent-border);
  border-radius: 0.85rem;
  background: var(--fas-accent);
  color: #fff;
  font-size: 0.88rem;
  font-weight: 700;
  padding: 0.8rem 1rem;
  cursor: pointer;
}
.login-submit:hover {
  background: var(--fas-accent-hover);
}
.login-submit:disabled {
  opacity: 0.6;
  cursor: wait;
}
.login-form-wrap :is(input, button, a):focus-visible {
  outline: 2px solid var(--fas-accent);
  outline-offset: 3px;
}
.login-error {
  margin: 0;
  padding: 0.8rem;
  border: 1px solid var(--fas-danger);
  border-radius: 0.8rem;
  background: var(--fas-danger-soft);
  color: var(--fas-danger);
  font-size: 0.8rem;
}
.login-footnote {
  border-top: 1px solid var(--fas-border);
  padding-top: 1.25rem;
  margin-top: 1.8rem;
  font-size: 0.72rem;
  color: var(--fas-muted);
  line-height: 1.7;
}
@media (max-width: 800px) {
  .login-shell {
    grid-template-columns: 1fr;
    max-width: 520px;
  }
  .login-brand {
    border-right: 0;
    border-bottom: 1px solid var(--fas-border);
    gap: 0;
  }
  .brand-story,
  .brand-footer {
    display: none;
  }
  .login-back {
    margin-bottom: 1.6rem;
  }
  .login-form-panel {
    padding: 1.5rem;
  }
  .fas-login {
    padding: 1rem;
  }
  .login-form-wrap h2 {
    font-size: 1.75rem;
  }
}
/* Premium auth surface: layered color and depth, with no flat black panel. */
.fas-login {
  background:
    radial-gradient(
      ellipse at 85% 15%,
      var(--fas-accent-soft),
      transparent 50%
    ),
    linear-gradient(
      140deg,
      var(--fas-bg),
      color-mix(in srgb, var(--fas-accent) 9%, var(--fas-elevated))
    );
  padding: clamp(1rem, 3vw, 3rem);
}
.login-shell {
  width: min(1240px, 100%);
  border-radius: 30px;
  grid-template-columns: 1.05fr 1fr;
  box-shadow: 0 25px 85px color-mix(in srgb, var(--fas-accent) 12%, transparent);
}
.login-brand {
  padding: 2.2rem 2.5rem 1.6rem;
  gap: 1.8rem;
  background:
    radial-gradient(
      ellipse at 60% 60%,
      var(--fas-accent-soft),
      transparent 65%
    ),
    linear-gradient(
      145deg,
      color-mix(in srgb, var(--fas-accent) 15%, var(--fas-elevated)),
      color-mix(in srgb, var(--fas-accent) 4%, var(--fas-card))
    );
}
.brand-story h1 {
  font-size: clamp(1.8rem, 3vw, 2.8rem);
  max-width: 17ch;
  margin: 0.7rem 0;
  line-height: 1.12;
}
.brand-eyebrow {
  font-size: 0.62rem;
  letter-spacing: 0.13em;
}
.brand-description {
  font-size: 0.8rem;
  line-height: 1.8;
  max-width: 370px;
  margin-bottom: 0;
}
.login-form-panel {
  position: relative;
  isolation: isolate;
  overflow: hidden;
  background:
    radial-gradient(ellipse at 95% 0, var(--fas-accent-soft), transparent 55%),
    linear-gradient(
      160deg,
      color-mix(in srgb, var(--fas-text) 4%, var(--fas-card)),
      color-mix(in srgb, var(--fas-accent) 10%, var(--fas-elevated))
    );
  padding: 2.5rem;
}
.login-form-panel:before {
  content: '';
  position: absolute;
  z-index: -1;
  inset: 25% -30% auto auto;
  width: 320px;
  height: 320px;
  border: 1px solid var(--fas-accent-border);
  border-radius: 50%;
  box-shadow:
    0 0 0 45px color-mix(in srgb, var(--fas-accent) 3%, transparent),
    0 0 0 90px color-mix(in srgb, var(--fas-accent) 2%, transparent);
  opacity: 0.4;
  pointer-events: none;
}
.login-form-wrap {
  padding: 1.7rem;
  border: 1px solid var(--fas-border-strong);
  border-radius: 24px;
  background: color-mix(in srgb, var(--fas-card) 88%, transparent);
  backdrop-filter: blur(18px);
  box-shadow: var(--fas-shadow-card);
  max-width: 440px;
}
.login-back {
  margin-bottom: 2rem;
  font-size: 0.72rem;
}
.login-form-wrap h2 {
  font-size: 1.9rem;
}
.login-description {
  font-size: 0.8rem;
}
.login-submit {
  border-radius: 50px;
  padding-inline: 1.2rem;
}
.login-footnote {
  font-size: 0.66rem;
}
.login-eyebrow {
  font-size: 0.63rem;
  letter-spacing: 0.08em;
}
@media (max-width: 900px) {
  .login-shell {
    grid-template-columns: 1fr;
    max-width: 600px;
  }
  .login-brand {
    padding: 1.4rem 1.8rem;
  }
  .brand-story {
    display: none;
  }
  .brand-footer {
    display: none;
  }
  .login-form-panel {
    padding: 1.5rem;
  }
  .brand-heading p {
    font-size: 0.6rem;
  }
  .login-form-wrap {
    max-width: none;
  }
  .login-back {
    margin-bottom: 1.4rem;
  }
}
@media (max-width: 480px) {
  .fas-login {
    padding: 0.7rem;
  }
  .login-shell {
    border-radius: 22px;
  }
  .login-form-panel {
    padding: 1rem;
  }
  .login-form-wrap {
    padding: 1.3rem;
    border-radius: 18px;
  }
  .login-form-wrap h2 {
    font-size: 1.65rem;
  }
}
</style>
