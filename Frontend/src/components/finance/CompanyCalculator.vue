<script setup>
import { nextTick, ref } from 'vue'
import { calculateCompanyExpression } from '@/utils/companyCalculator'
const emit = defineEmits(['use', 'toggle'])
const open = ref(false),
  expression = ref(''),
  answer = ref(''),
  error = ref(''),
  history = ref([]),
  input = ref(null),
  toggle = ref(null),
  copied = ref(false)
const keys = [
  '(',
  ')',
  '%',
  '⌫',
  '7',
  '8',
  '9',
  '÷',
  '4',
  '5',
  '6',
  '×',
  '1',
  '2',
  '3',
  '−',
  '0',
  '.',
  'C',
  '+',
]
const show = async () => {
  open.value = !open.value
  emit('toggle', open.value)
  if (open.value) {
    await nextTick()
    input.value?.focus()
  } else toggle.value?.focus()
}
const press = (key) => {
  if (answer.value && /[0-9.(]/.test(key)) expression.value = ''
  else if (answer.value && ['+', '−', '×', '÷', '%'].includes(key))
    expression.value = answer.value
  error.value = ''
  answer.value = ''
  copied.value = false
  if (key === 'C') expression.value = ''
  else if (key === '⌫') expression.value = expression.value.slice(0, -1)
  else if (expression.value.length < 160) expression.value += key
}
const calculate = () => {
  error.value = ''
  copied.value = false
  try {
    answer.value = calculateCompanyExpression(expression.value)
    history.value = [
      { expression: expression.value, answer: answer.value },
      ...history.value,
    ].slice(0, 6)
  } catch (failure) {
    answer.value = ''
    error.value = failure.message
  }
}
const copy = async () => {
  try {
    await navigator.clipboard.writeText(answer.value)
    copied.value = true
  } catch {
    error.value =
      'Clipboard tidak tersedia. Pilih lalu salin hasil yang tampil.'
  }
}
const clearResult = () => {
  answer.value = ''
  error.value = ''
  copied.value = false
}
const restoreHistory = (item) => {
  expression.value = item.expression
  answer.value = item.answer
  error.value = ''
  copied.value = false
}
</script>
<template>
  <aside
    class="company-calculator"
    :class="{ expanded: open }"
    aria-label="Kalkulator perusahaan"
  >
    <button
      ref="toggle"
      class="calculator-toggle"
      :aria-label="open ? 'Sembunyikan kalkulator' : 'Buka kalkulator'"
      :title="open ? 'Sembunyikan kalkulator' : 'Buka kalkulator'"
      :aria-expanded="open"
      aria-controls="company-calculator-panel"
      @click="show"
    >
      <svg
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="1.8"
        width="20"
        height="20"
        aria-hidden="true"
      >
        <rect x="5" y="2" width="14" height="20" rx="3" />
        <path d="M8 6h8M8 10h2m4 0h2m-8 4h2m4 0h2m-8 4h2m4 0h2" /></svg
      ><span>{{ open ? 'Sembunyikan kalkulator' : 'Buka kalkulator' }}</span>
    </button>
    <section
      v-if="open"
      id="company-calculator-panel"
      class="calculator-body"
      @keydown.esc.stop.prevent="show"
    >
      <h2>Kalkulator</h2>
      <p class="muted">
        Hitung nominal pengeluaran. Persen berarti dibagi 100; contoh 200000 ×
        10%.
      </p>
      <label
        >Perhitungan<input
          ref="input"
          v-model="expression"
          maxlength="160"
          inputmode="text"
          autocomplete="off"
          spellcheck="false"
          placeholder="2000000 + 500000"
          @input="clearResult"
          @keydown.enter.prevent="calculate"
      /></label>
      <output class="calculator-result" aria-live="polite">{{
        answer || '0.00'
      }}</output>
      <p v-if="error" class="message error" role="alert">{{ error }}</p>
      <div class="calculator-keys">
        <button
          v-for="key in keys"
          :key="key"
          type="button"
          :aria-label="
            key === '⌫'
              ? 'Hapus satu karakter'
              : key === 'C'
                ? 'Bersihkan'
                : key
          "
          @click="press(key)"
        >
          {{ key }}</button
        ><button class="primary equals" @click="calculate">= Hitung</button>
      </div>
      <div class="calculator-actions">
        <button :disabled="!answer" @click="copy">
          {{ copied ? 'Tersalin' : 'Salin hasil' }}</button
        ><button
          class="primary"
          :disabled="
            !answer ||
            answer.startsWith('-') ||
            answer.split('.')[0].length > 15
          "
          @click="emit('use', answer)"
        >
          Pakai nominal
        </button>
      </div>
      <small
        >Pakai nominal mengisi komponen laba yang dipilih. Simpan dasar
        perhitungan untuk menyimpan perubahan. Persen pada kalkulator berarti
        dibagi 100.</small
      >
      <div v-if="history.length" class="calculator-history">
        <h3>Hitungan sesi ini</h3>
        <button
          v-for="(item, index) in history"
          :key="index"
          @click="restoreHistory(item)"
        >
          <span>{{ item.expression }}</span
          ><strong>{{ item.answer }}</strong>
        </button>
      </div>
    </section>
  </aside>
</template>
<style scoped>
.company-calculator {
  min-width: 0;
  position: sticky;
  top: 7rem;
  align-self: start;
  border: 1px solid var(--fas-border);
  border-radius: 1.1rem;
  background: var(--fas-card);
  overflow: hidden;
}
.calculator-toggle {
  width: 100%;
  border: 0 !important;
  border-radius: 0 !important;
  justify-content: flex-start !important;
  color: var(--fas-accent-text) !important;
  background: var(--fas-accent-soft) !important;
}
.company-calculator:not(.expanded) .calculator-toggle span {
  display: none;
}
.company-calculator:not(.expanded) .calculator-toggle {
  padding: 0.6rem !important;
  justify-content: center !important;
}
.calculator-body {
  padding: 1rem;
  max-height: calc(100dvh - 11rem);
  overflow-y: auto;
  overscroll-behavior: contain;
}
.calculator-body p {
  font-size: 0.75rem;
  margin: 0.5rem 0 1rem;
}
.calculator-result {
  display: block;
  text-align: right;
  font-size: 1.55rem;
  font-weight: 750;
  font-variant-numeric: tabular-nums;
  overflow-wrap: anywhere;
  padding: 1rem 0;
  color: var(--fas-text);
  user-select: text;
}
.calculator-keys {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 0.45rem;
}
.calculator-keys button {
  padding: 0.5rem !important;
  font-size: 1rem !important;
}
.calculator-keys .equals {
  grid-column: 1/-1;
}
.calculator-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.5rem;
  margin-top: 0.7rem;
}
.calculator-actions button {
  padding: 0.5rem !important;
}
.calculator-history {
  margin-top: 1rem;
  border-top: 1px solid var(--fas-border);
  padding-top: 0.8rem;
}
.calculator-history h3 {
  font-size: 0.75rem;
  font-weight: 700;
  margin-bottom: 0.4rem;
}
.calculator-history button {
  display: flex !important;
  flex-direction: column;
  align-items: stretch !important;
  width: 100%;
  margin-top: 0.4rem;
  overflow-wrap: anywhere;
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.calculator-history span {
  color: var(--fas-muted);
  font-size: 0.72rem;
}
@media (max-width: 1199px) {
  .company-calculator:not(.expanded) .calculator-toggle span {
    display: inline;
  }
  .company-calculator:not(.expanded) .calculator-toggle {
    justify-content: flex-start !important;
  }
  .company-calculator {
    position: static;
    grid-row: 1;
  }
  .calculator-body {
    max-height: 32rem;
  }
  .calculator-keys {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
</style>
