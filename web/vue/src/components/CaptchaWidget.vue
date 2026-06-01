<template>
  <div class="flex flex-col" style="width: fit-content; max-width: 92vw">
    <!-- CLICK -->
    <template v-if="mode === 'click'">
      <div v-if="thumbImage" class="inline-flex items-center bg-gray-50 border rounded mb-2 px-2 py-1">
        <img :src="thumbImage" alt="thumb" style="display:block;max-height:52px" />
      </div>
      <div v-if="masterImage" class="flex items-center justify-center bg-gray-50 border rounded p-1.5">
        <div style="display:inline-block;line-height:0;position:relative">
          <img
            ref="clickImg"
            :src="masterImage"
            alt="captcha"
            style="border-radius:6px;cursor:crosshair;display:block;max-height:380px;max-width:100%;object-fit:contain"
            @load="onMasterLoad"
            @click="onClickMaster"
          />
          <span
            v-for="(p, i) in (natural.w > 0 ? clicks : [])"
            :key="i"
            class="flex items-center justify-center text-white font-bold"
            :style="dotStyle(p)"
          >{{ i + 1 }}</span>
        </div>
      </div>
    </template>

    <!-- SLIDE -->
    <template v-else-if="mode === 'slide'">
      <div v-if="masterImage" class="bg-gray-50 border rounded p-1.5">
        <div ref="slideWrap" style="display:inline-block;line-height:0;position:relative">
          <img :src="masterImage" alt="captcha" style="border-radius:6px;display:block;max-width:100%" @load="onMasterLoad" />
          <img
            v-if="thumbImage"
            :src="thumbImage"
            alt="tile"
            :style="tileStyle"
          />
        </div>
      </div>
      <div class="mt-3">
        <a-slider :min="0" :max="slideMax" v-model:value="slideX" :tip-formatter="null" @change="touched = true" />
        <div class="text-xs text-gray-500 text-center">{{ t('auth.login.captcha_slide_hint') }}</div>
      </div>
    </template>

    <!-- ROTATE -->
    <template v-else-if="mode === 'rotate'">
      <div v-if="masterImage" class="flex items-center justify-center bg-gray-50 border rounded p-2">
        <div style="position:relative;line-height:0">
          <img :src="masterImage" alt="captcha" style="border-radius:50%;display:block;max-width:100%" @load="onMasterLoad" />
          <img
            v-if="thumbImage"
            :src="thumbImage"
            alt="thumb"
            :style="rotateThumbStyle"
          />
        </div>
      </div>
      <div class="mt-3">
        <a-slider :min="0" :max="360" v-model:value="rotateAngle" :tip-formatter="(v) => v + '°'" @change="touched = true" />
        <div class="text-xs text-gray-500 text-center">{{ t('auth.login.captcha_rotate_hint') }}</div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue';
import { useI18n } from 'vue-i18n';

const { t } = useI18n();

const props = defineProps({
  // challenge: { mode, master_image, thumb_image, dot_num, tile_x, tile_y, tile_width, tile_height, thumb_size }
  challenge: { type: Object, default: () => ({}) },
});
const emit = defineEmits(['update:answer', 'update:ready']);

const mode = computed(() => props.challenge?.mode || 'click');
const masterImage = computed(() => props.challenge?.master_image || '');
const thumbImage = computed(() => props.challenge?.thumb_image || '');
const dotNum = computed(() => Number(props.challenge?.dot_num) || 0);

const natural = reactive({ w: 0, h: 0 });
const displayW = ref(0);

// click
const clicks = ref([]);
// slide
const slideX = ref(0);
// rotate
const rotateAngle = ref(0);
// Whether the user has interacted (dragged the slider). Slide/rotate readiness
// is based on this, not on slideX>0 (which is fragile: true-by-default when the
// tile starts at x>0, or never-true if the slider can't move before image load).
const touched = ref(false);

function resetForChallenge() {
  natural.w = 0;
  natural.h = 0;
  clicks.value = [];
  slideX.value = Number(props.challenge?.tile_x) || 0;
  rotateAngle.value = 0;
  touched.value = false;
}
watch(() => props.challenge, resetForChallenge, { immediate: true });

function onMasterLoad(ev) {
  natural.w = ev.target.naturalWidth;
  natural.h = ev.target.naturalHeight;
  displayW.value = ev.target.clientWidth || ev.target.naturalWidth;
}

// ---- click ----
function onClickMaster(e) {
  if (mode.value !== 'click') return;
  if (!dotNum.value || clicks.value.length >= dotNum.value) return;
  const rect = e.currentTarget.getBoundingClientRect();
  // Map displayed px back to natural px (image may be scaled by max-width).
  const scale = rect.width > 0 ? natural.w / rect.width : 1;
  const x = Math.round((e.clientX - rect.left) * scale);
  const y = Math.round((e.clientY - rect.top) * scale);
  clicks.value = [...clicks.value, { x, y }];
  touched.value = true;
}
function dotStyle(p) {
  return {
    position: 'absolute', width: '22px', height: '22px', borderRadius: '50%',
    background: 'rgba(37,99,235,0.22)', border: '2px solid #2563eb', fontSize: '12px',
    left: (p.x / natural.w) * 100 + '%',
    top: (p.y / natural.h) * 100 + '%',
    transform: 'translate(-50%, -50%)',
  };
}

// ---- slide ----
const tileW = computed(() => Number(props.challenge?.tile_width) || 0);
const tileY = computed(() => Number(props.challenge?.tile_y) || 0);
const slideMax = computed(() => Math.max(0, (natural.w || 0) - tileW.value));
const tileStyle = computed(() => ({
  position: 'absolute',
  left: natural.w > 0 ? (slideX.value / natural.w) * 100 + '%' : '0',
  top: natural.h > 0 ? (tileY.value / natural.h) * 100 + '%' : '0',
  width: natural.w > 0 ? (tileW.value / natural.w) * 100 + '%' : 'auto',
  height: 'auto',
}));

// ---- rotate ----
const rotateThumbStyle = computed(() => {
  const sz = Number(props.challenge?.thumb_size) || 0;
  return {
    position: 'absolute',
    left: '50%',
    top: '50%',
    width: natural.w > 0 && sz > 0 ? (sz / natural.w) * 100 + '%' : '50%',
    transform: `translate(-50%, -50%) rotate(${rotateAngle.value}deg)`,
    borderRadius: '50%',
  };
});

// Emit answer + readiness for the active mode.
const answer = computed(() => {
  if (mode.value === 'slide') return { x: Math.round(slideX.value), y: Math.round(tileY.value) };
  if (mode.value === 'rotate') return { angle: Math.round(rotateAngle.value) };
  return clicks.value;
});
const ready = computed(() => {
  if (mode.value === 'slide' || mode.value === 'rotate') return touched.value;
  return dotNum.value > 0 && clicks.value.length === dotNum.value;
});

watch(answer, (v) => emit('update:answer', v), { immediate: true, deep: true });
watch(ready, (v) => emit('update:ready', v), { immediate: true });

function clear() {
  clicks.value = [];
  slideX.value = Number(props.challenge?.tile_x) || 0;
  rotateAngle.value = 0;
  touched.value = false;
}

// Expose live state so the parent can read it directly at submit time, avoiding
// any v-model/emit timing issues.
defineExpose({ clear, isReady: () => ready.value, getAnswer: () => answer.value });
</script>
