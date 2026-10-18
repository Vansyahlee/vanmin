<script setup>
import { computed, reactive, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useRouter } from 'vue-router'
import FasSelect from '@/components/ui/FasSelect.vue'
import { useModalDialog } from '@/composables/useModalDialog'
import {
  useFinanceLedger,
  financeDate,
  financeError,
} from '@/composables/useFinanceLedger'
import {
  getUsers,
  createUser,
  updateUser,
  resetUserPassword,
  getUserHistory,
} from '@/services/user'
import {
  downloadFinanceXlsx,
  openFinancePrint,
  writeFinancePrint,
} from '@/utils/financeDocuments'
import '@/assets/finance-ledger.css'

const auth = useAuthStore(),
  router = useRouter()
const roleName = (role) =>
  ({ admin: 'Administrator', finance: 'Finance', pegawai: 'Pegawai Gudang' })[
    role
  ] || role
const roleCards = [
  {
    role: 'admin',
    description: 'Mengelola akun, operasional, dan laporan perusahaan.',
  },
  {
    role: 'finance',
    description: 'Mengelola faktur, piutang, pembayaran, dan laporan keuangan.',
  },
  {
    role: 'pegawai',
    description: 'Mengelola pesanan, stok, picking, packing, dan distribusi.',
  },
]
const {
  filters,
  applied,
  data,
  summary,
  total,
  loading,
  exporting,
  ready,
  error,
  page,
  load,
  report,
  canPrevious,
  canNext,
} = useFinanceLedger(
  getUsers,
  { query: '', role: '', status: '' },
  {
    title: 'PT FAS — Daftar Akun Internal',
    filename: 'PTFAS_Akun_Internal.xlsx',
    headings: ['Nama', 'Email', 'Peran', 'Status', 'Dibuat'],
    rows: (items) =>
      items.map((x) => [
        x.name,
        x.email,
        roleName(x.role),
        x.is_active ? 'Aktif' : 'Nonaktif',
        financeDate(x.created_at),
      ]),
    description: () =>
      'Daftar akun internal sesuai filter. Tidak memuat password.',
  },
)
const dialog = ref(''),
  selected = ref(null),
  busy = ref(false),
  formError = ref(''),
  message = ref(''),
  history = ref([]),
  showPassword = ref(false),
  confirmed = ref(false)
const form = reactive({
  name: '',
  email: '',
  role: 'pegawai',
  status: 'active',
  password: '',
  revision: 0,
})
const heading = computed(
  () =>
    ({
      create: 'Tambah Akun Internal',
      edit: 'Edit Akun',
      password: 'Reset Password',
      history: 'Riwayat Perubahan Akun',
    })[dialog.value],
)
const close = () => {
  if (busy.value) return
  dialog.value = ''
  form.password = ''
  showPassword.value = false
  confirmed.value = false
  formError.value = ''
}
useModalDialog(() => (dialog.value ? '#user-dialog' : ''), close)
const open = (mode, item = null) => {
  selected.value = item
  formError.value = ''
  message.value = ''
  showPassword.value = false
  confirmed.value = false
  history.value = []
  Object.assign(form, {
    name: item?.name || '',
    email: item?.email || '',
    role: item?.role || 'pegawai',
    status: item?.is_active === false ? 'inactive' : 'active',
    password: '',
    revision: item?.revision ?? 0,
  })
  dialog.value = mode
}
const save = async () => {
  if (busy.value) return
  formError.value = ''
  if (
    (dialog.value === 'create' || dialog.value === 'password') &&
    ([...form.password].length < 8 ||
      new TextEncoder().encode(form.password).length > 72)
  ) {
    formError.value = 'Password minimal 8 karakter dan maksimal 72 byte.'
    return
  }
  if (dialog.value === 'password' && !confirmed.value) {
    formError.value = 'Konfirmasi reset password terlebih dahulu.'
    return
  }
  busy.value = true
  try {
    let result
    if (dialog.value === 'password')
      result = await resetUserPassword(selected.value.id, {
        password: form.password,
        revision: form.revision,
      })
    else {
      const payload = {
        name: form.name.trim(),
        email: form.email.trim(),
        role: form.role,
        is_active: form.status === 'active',
        revision: form.revision,
      }
      if (dialog.value === 'create') payload.password = form.password
      result =
        dialog.value === 'create'
          ? await createUser(payload)
          : await updateUser(selected.value.id, payload)
    }
    const mode = dialog.value
    busy.value = false
    close()
    if (result.reauthenticate) {
      auth.logout()
      await router.replace('/login')
      return
    }
    message.value =
      mode === 'password'
        ? 'Password diperbarui. Sesi lama pengguna tidak berlaku lagi. Sampaikan password melalui jalur internal yang aman.'
        : 'Akun tersimpan. Perubahan peran/status akan berlaku pada permintaan berikutnya.'
    await load()
  } catch (e) {
    formError.value = financeError(e)
  } finally {
    busy.value = false
  }
}
const viewHistory = async (item) => {
  open('history', item)
  busy.value = true
  try {
    history.value = (await getUserHistory(item.id)).data || []
  } catch (e) {
    formError.value = financeError(e)
  } finally {
    busy.value = false
  }
}
const actionName = (value) =>
  ({
    CREATE: 'Akun dibuat',
    UPDATE: 'Profil/peran/status diubah',
    RESET_PASSWORD: 'Password direset',
  })[value] || value
const historyRows = computed(() =>
  history.value.map((x) => [
    financeDate(x.created_at),
    x.actor_name,
    actionName(x.action),
    x.before_data
      ? `${x.before_data.name} · ${roleName(x.before_data.role)} · ${x.before_data.is_active ? 'Aktif' : 'Nonaktif'}`
      : '—',
    `${x.after_data.name} · ${roleName(x.after_data.role)} · ${x.after_data.is_active ? 'Aktif' : 'Nonaktif'}`,
  ]),
)
const exportHistory = (kind) => {
  try {
    const title = 'Riwayat akun — ' + selected.value.name,
      description =
        'Maksimal 50 perubahan terakhir. Password tidak disertakan.',
      headings = ['Waktu', 'Petugas', 'Perubahan', 'Sebelum', 'Sesudah']
    if (kind === 'excel')
      downloadFinanceXlsx('PTFAS_Riwayat_Akun.xlsx', [
        [title],
        [description],
        headings,
        ...historyRows.value,
      ])
    else
      writeFinancePrint(
        openFinancePrint(),
        title,
        description,
        headings,
        historyRows.value,
      )
  } catch (e) {
    formError.value = financeError(e)
  }
}
</script>
<template>
  <main class="finance-ledger users-page">
    <header class="page-heading">
      <div>
        <p class="eyebrow">Administration · Users</p>
        <h1>Akun & Hak Akses</h1>
        <p class="muted">
          Satu akun untuk setiap pegawai. Atur akses sesuai tanggung jawabnya.
        </p>
      </div>
      <button class="primary" :disabled="exporting" @click="open('create')">
        + Tambah Akun
      </button>
    </header>
    <section class="cards">
      <article class="card">
        <span>Akun sesuai filter</span><strong>{{ ready ? total : '—' }}</strong
        ><small>Seluruh halaman hasil pencarian.</small>
      </article>
      <article class="card">
        <span>Aktif / Nonaktif</span
        ><strong
          >{{ ready ? summary.active : '—' }} /
          {{ ready ? summary.inactive : '—' }}</strong
        ><small>Akun nonaktif tidak bisa mengakses sistem.</small>
      </article>
      <article class="card">
        <span>Administrator aktif sesuai filter</span
        ><strong>{{ ready ? summary.admins : '—' }}</strong
        ><small>Akun admin terakhir tetap dilindungi.</small>
      </article>
    </section>
    <details class="panel roles-guide">
      <summary>Panduan peran pengguna</summary>
      <div class="role-cards">
        <article v-for="item in roleCards" :key="item.role">
          <span class="badge">{{ roleName(item.role) }}</span>
          <p>{{ item.description }}</p>
        </article>
      </div>
      <small
        >Daftar akun bukan daftar payroll. Penggajian tetap dicatat melalui
        Finance → Pengeluaran.</small
      >
    </details>
    <p v-if="error" class="message error" role="alert">{{ error }}</p>
    <p v-if="message" class="message" role="status">{{ message }}</p>
    <form class="panel" @submit.prevent="load(true)">
      <fieldset :disabled="loading || exporting">
        <div class="filters">
          <label
            >Cari akun<input
              v-model="filters.query"
              type="search"
              maxlength="200"
              placeholder="Nama atau email pegawai" /></label
          ><label
            >Peran<FasSelect v-model="filters.role" aria-label="Filter peran"
              ><option value="">Semua peran</option>
              <option
                v-for="item in roleCards"
                :key="item.role"
                :value="item.role"
              >
                {{ roleName(item.role) }}
              </option></FasSelect
            ></label
          ><label
            >Status<FasSelect
              v-model="filters.status"
              aria-label="Filter status akun"
              ><option value="">Semua status</option>
              <option value="active">Aktif</option>
              <option value="inactive">Nonaktif</option></FasSelect
            ></label
          >
          <div class="actions">
            <button type="submit" class="primary">Terapkan Filter</button>
          </div>
        </div>
      </fieldset>
    </form>
    <section class="panel">
      <header class="panel-heading">
        <div>
          <h2>Daftar Akun Internal</h2>
          <small
            >Cari: {{ applied.query || 'Semua' }} · Peran:
            {{ applied.role ? roleName(applied.role) : 'Semua' }} · Status:
            {{
              applied.status === 'active'
                ? 'Aktif'
                : applied.status === 'inactive'
                  ? 'Nonaktif'
                  : 'Semua'
            }}</small
          >
        </div>
        <div class="actions">
          <button :disabled="loading || exporting" @click="load()">
            Perbarui</button
          ><button
            :disabled="!ready || loading || exporting"
            @click="report('excel')"
          >
            Excel</button
          ><button
            :disabled="!ready || loading || exporting"
            @click="report('print')"
          >
            Print / PDF
          </button>
        </div>
      </header>
      <div class="table-scroll">
        <table>
          <thead>
            <tr>
              <th>Pengguna</th>
              <th>Peran</th>
              <th>Status</th>
              <th>Dibuat</th>
              <th>Aksi</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in data" :key="item.id">
              <td>
                <strong>{{ item.name }}</strong
                ><span v-if="item.id === auth.user?.id" class="badge self-badge"
                  >Akun saya</span
                ><small>{{ item.email }}</small>
              </td>
              <td>{{ roleName(item.role) }}</td>
              <td>
                <span class="badge" :class="{ void: !item.is_active }">{{
                  item.is_active ? 'Aktif' : 'Nonaktif'
                }}</span>
              </td>
              <td>{{ financeDate(item.created_at) }}</td>
              <td>
                <div class="actions">
                  <button :disabled="exporting" @click="open('edit', item)">
                    Edit</button
                  ><button
                    :disabled="exporting || item.id === auth.user?.id"
                    @click="open('password', item)"
                  >
                    Reset Password</button
                  ><button :disabled="exporting" @click="viewHistory(item)">
                    Riwayat
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="!data.length">
              <td colspan="5" class="empty">
                {{ loading ? 'Memuat akun…' : 'Tidak ada akun sesuai filter.' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <footer class="pagination">
        <span>{{ total }} akun · Halaman {{ page + 1 }}</span>
        <div class="actions">
          <button :disabled="!canPrevious" @click="load(false, page - 1)">
            Sebelumnya</button
          ><button :disabled="!canNext" @click="load(false, page + 1)">
            Berikutnya
          </button>
        </div>
      </footer>
    </section>
    <Teleport to="body"
      ><div
        v-if="dialog"
        class="finance-ledger dialog-backdrop users-page"
        @click.self="close"
      >
        <section
          id="user-dialog"
          class="dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="user-dialog-title"
        >
          <header>
            <div>
              <h2 id="user-dialog-title">{{ heading }}</h2>
              <small v-if="selected"
                >{{ selected.name }} · {{ selected.email }}</small
              >
            </div>
            <button :disabled="busy" @click="close">Tutup</button>
          </header>
          <form v-if="dialog !== 'history'" @submit.prevent="save">
            <div class="body">
              <p v-if="formError" class="message error" role="alert">
                {{ formError }}
              </p>
              <fieldset :disabled="busy">
                <div class="form-grid">
                  <template v-if="dialog !== 'password'"
                    ><label
                      >Nama lengkap *<input
                        v-model="form.name"
                        required
                        maxlength="150"
                        autocomplete="off" /></label
                    ><label
                      >Email akun *<input
                        v-model="form.email"
                        required
                        type="email"
                        maxlength="254"
                        autocomplete="off" /></label
                    ><label
                      >Peran *<FasSelect
                        v-model="form.role"
                        aria-label="Peran akun"
                        :disabled="busy || selected?.id === auth.user?.id"
                        ><option
                          v-for="item in roleCards"
                          :key="item.role"
                          :value="item.role"
                        >
                          {{ roleName(item.role) }}
                        </option></FasSelect
                      ></label
                    ><label
                      >Status akun *<FasSelect
                        v-model="form.status"
                        aria-label="Status akun"
                        :disabled="busy || selected?.id === auth.user?.id"
                        ><option value="active">Aktif</option>
                        <option value="inactive">Nonaktif</option></FasSelect
                      ></label
                    ></template
                  >
                  <label
                    v-if="dialog === 'create' || dialog === 'password'"
                    class="span-two"
                    >{{
                      dialog === 'create' ? 'Password awal' : 'Password baru'
                    }}
                    *<span class="user-password"
                      ><input
                        v-model="form.password"
                        :type="showPassword ? 'text' : 'password'"
                        required
                        minlength="8"
                        maxlength="72"
                        autocomplete="new-password" /><button
                        type="button"
                        :aria-label="
                          showPassword
                            ? 'Sembunyikan password'
                            : 'Tampilkan password'
                        "
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
                    ><small
                      >Minimal 8 karakter, maksimal 72 byte. Password tidak
                      ditampilkan kembali setelah disimpan.</small
                    ></label
                  >
                </div>
                <label v-if="dialog === 'password'" class="reset-confirm"
                  ><input v-model="confirmed" type="checkbox" required />Saya
                  mengonfirmasi reset password dan penghentian sesi lama
                  pengguna ini.</label
                >
                <p v-if="dialog === 'edit'" class="muted">
                  Perubahan akun akan menghentikan sesi lamanya. Jika mengedit
                  akun sendiri, Anda perlu masuk kembali.
                </p>
              </fieldset>
            </div>
            <footer>
              <button type="button" :disabled="busy" @click="close">
                Batal</button
              ><button class="primary" type="submit" :disabled="busy">
                {{
                  busy
                    ? 'Menyimpan…'
                    : dialog === 'password'
                      ? 'Konfirmasi Reset Password'
                      : 'Simpan Akun'
                }}
              </button>
            </footer>
          </form>
          <div v-else class="body">
            <p v-if="formError" class="message error" role="alert">
              {{ formError }}
            </p>
            <div class="actions">
              <button
                :disabled="busy || !history.length"
                @click="exportHistory('excel')"
              >
                Excel riwayat</button
              ><button
                :disabled="busy || !history.length"
                @click="exportHistory('print')"
              >
                Print / PDF riwayat
              </button>
            </div>
            <div v-if="!history.length" class="empty">
              {{
                busy
                  ? 'Memuat riwayat…'
                  : 'Belum ada perubahan sejak fitur ini diaktifkan.'
              }}
            </div>
            <ol class="audit-list">
              <li v-for="item in history" :key="item.id">
                <span class="badge">{{ actionName(item.action) }}</span>
                <p>
                  {{ item.actor_name }} · {{ financeDate(item.created_at) }}
                </p>
                <small v-if="item.before_data"
                  >Sebelum: {{ roleName(item.before_data.role) }} ·
                  {{ item.before_data.is_active ? 'Aktif' : 'Nonaktif' }} ·
                  {{ item.before_data.email }}</small
                ><small
                  >Sesudah: {{ roleName(item.after_data.role) }} ·
                  {{ item.after_data.is_active ? 'Aktif' : 'Nonaktif' }} ·
                  {{ item.after_data.email }}</small
                >
              </li>
            </ol>
          </div>
        </section>
      </div></Teleport
    >
  </main>
</template>
<style scoped>
.roles-guide {
  padding: 1.1rem;
}
.roles-guide summary {
  cursor: pointer;
  font-weight: 700;
}
.role-cards {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1rem;
  margin: 1rem 0;
}
.role-cards article {
  padding: 1rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.9rem;
  background: var(--fas-elevated);
}
.role-cards p {
  font-size: 0.8rem;
  color: var(--fas-muted);
}
.self-badge {
  margin-left: 0.5rem;
}
.user-password {
  position: relative;
  display: block;
}
.user-password input {
  padding-right: 3.4rem;
}
.user-password input::-ms-reveal,
.user-password input::-ms-clear {
  display: none;
}
.user-password button {
  position: absolute;
  right: 0.3rem;
  top: 0.3rem;
  min-height: 2.2rem;
  width: 2.4rem;
  padding: 0.4rem;
  border: 0;
  background: transparent;
}
.user-password svg {
  width: 1.2rem;
  height: 1.2rem;
}
.reset-confirm {
  display: flex !important;
  flex-direction: row !important;
  align-items: flex-start;
  gap: 0.7rem;
  margin-top: 1.2rem;
}
.reset-confirm input {
  width: 1.1rem;
  min-height: 1.1rem;
  height: 1.1rem;
  margin-top: 0.2rem;
}
.audit-list {
  list-style: none;
  padding: 0;
  display: grid;
  gap: 1rem;
  margin-top: 1rem;
}
.audit-list li {
  padding: 1rem;
  border: 1px solid var(--fas-border);
  border-radius: 0.85rem;
  background: var(--fas-card);
}
.audit-list p {
  font-size: 0.8rem;
}
@media (max-width: 700px) {
  .role-cards {
    grid-template-columns: 1fr;
  }
}
</style>
