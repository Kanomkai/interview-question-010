<template>
  <div>
    <div class="card" style="text-align:center;">
      <h2 style="font-size:1.4rem;margin-bottom:4px;">ผลการสอบ</h2>
      <p style="color:var(--muted);margin-bottom:24px;">{{ result.examinee }}</p>

      <div class="score-circle">
        <span class="score-num">{{ result.score }}</span>
        <span class="score-denom">/ {{ result.total }}</span>
      </div>

      <div style="font-size:1.15rem;font-weight:700;margin-bottom:8px;">
        <span :style="{ color: passed ? 'var(--success)' : 'var(--danger)' }">
          {{ passed ? '🎉 ผ่านการสอบ' : '❌ ไม่ผ่านการสอบ' }}
        </span>
      </div>
      <p style="color:var(--muted);font-size:.9rem;">
        คะแนน {{ result.score }} จาก {{ result.total }} ({{ pct }}%)
      </p>
    </div>

    <div class="card">
      <h3 style="margin-bottom:20px;font-size:1.1rem;">เฉลยข้อสอบ</h3>
      <div
        v-for="(detail, idx) in result.details"
        :key="detail.question_id"
        style="margin-bottom:20px;padding-bottom:20px;border-bottom:1px solid var(--border);"
      >
        <div class="q-number">ข้อที่ {{ idx + 1 }}</div>
        <div class="q-body" style="font-size:1rem;">
          {{ getQuestion(detail.question_id)?.body }}
        </div>
        <div class="choice-list" style="margin-top:10px;">
          <div
            v-for="c in getQuestion(detail.question_id)?.choices"
            :key="c.id"
            class="choice-item"
            :class="{
              correct: c.id === detail.correct_id,
              wrong: c.id === detail.chosen_id && !detail.is_correct,
            }"
          >
            <div class="choice-radio">
              <div
                v-if="c.id === detail.correct_id || c.id === detail.chosen_id"
                class="choice-radio-dot"
              ></div>
            </div>
            <span>{{ c.body }}</span>
            <span v-if="c.id === detail.correct_id" style="margin-left:auto;font-size:.8rem;color:var(--success);">✓ เฉลย</span>
            <span v-else-if="c.id === detail.chosen_id && !detail.is_correct" style="margin-left:auto;font-size:.8rem;color:var(--danger);">✗ คำตอบของคุณ</span>
          </div>
        </div>
      </div>
    </div>

    <div class="card">
      <div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:16px;">
        <h3 style="font-size:1.1rem;">ประวัติการสอบ</h3>
        <button class="btn btn-outline" style="padding:6px 16px;font-size:.85rem;" @click="loadHistory">รีเฟรช</button>
      </div>

      <div v-if="histLoading" style="text-align:center;padding:20px;color:var(--muted);">กำลังโหลด...</div>
      <div v-else-if="history.length === 0" style="text-align:center;padding:20px;color:var(--muted);">ยังไม่มีประวัติ</div>
      <div v-else style="overflow-x:auto;">
        <table class="results-table">
          <thead>
            <tr>
              <th>#</th>
              <th>ชื่อผู้สอบ</th>
              <th>คะแนน</th>
              <th>ผล</th>
              <th>วันเวลา</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(r, i) in history"
              :key="r.id"
              :style="r.id === result.id ? 'background:#EFF6FF;' : ''"
            >
              <td style="color:var(--muted);">{{ i + 1 }}</td>
              <td style="font-weight:600;">{{ r.examinee }}</td>
              <td>{{ r.score }} / {{ r.total }}</td>
              <td>
                <span class="badge" :class="r.score / r.total >= 0.6 ? 'badge-pass' : 'badge-fail'">
                  {{ r.score / r.total >= 0.6 ? 'ผ่าน' : 'ไม่ผ่าน' }}
                </span>
              </td>
              <td style="color:var(--muted);font-size:.85rem;">{{ r.submitted_at }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div style="text-align:center;margin-top:8px;">
      <button class="btn btn-success" style="min-width:200px;" @click="$emit('retry')">
        🔄 สอบอีกครั้ง
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { fetchResults } from '../api.js'

const props = defineProps({
  result:    { type: Object, required: true },
  questions: { type: Array,  required: true },
})
defineEmits(['retry'])

const history     = ref([])
const histLoading = ref(true)

// Pass threshold is 60%
const passed = computed(() => props.result.score / props.result.total >= 0.6)
const pct    = computed(() => Math.round(props.result.score / props.result.total * 100))

function getQuestion(qid) {
  return props.questions.find(q => q.id === qid)
}

async function loadHistory() {
  histLoading.value = true
  try {
    history.value = await fetchResults()
  } catch {
    history.value = []
  } finally {
    histLoading.value = false
  }
}

onMounted(loadHistory)
</script>
