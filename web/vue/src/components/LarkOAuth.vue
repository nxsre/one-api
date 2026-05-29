<template>
  <div style="min-height: 300px; position: relative">
    <a-spin
      size="large"
      :tip="prompt"
      style="display: flex; justify-content: center; align-items: center; min-height: 300px"
    >
      <div style="min-height: 1px" />
    </a-spin>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { API, showError, showSuccess } from '../helpers';
import { useUserStore } from '../stores/user';

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();

const prompt = ref('处理中...');

async function sendCode(code, state, count) {
  const res = await API.get(`/api/oauth/lark?code=${code}&state=${state}`);
  const { success, message, data } = res.data;
  if (success) {
    if (message === 'bind') {
      showSuccess('绑定成功！');
      router.push('/setting');
    } else {
      userStore.login(data);
      showSuccess('登录成功！');
      router.push('/');
    }
  } else {
    showError(message);
    if (count === 0) {
      prompt.value = '操作失败，重定向至登录界面中...';
      router.push('/setting'); // in case this is failed to bind lark
      return;
    }
    count++;
    prompt.value = `出现错误，第 ${count} 次重试中...`;
    await new Promise((resolve) => setTimeout(resolve, count * 2000));
    await sendCode(code, state, count);
  }
}

onMounted(() => {
  const code = route.query.code;
  const state = route.query.state;
  sendCode(code, state, 0).then();
});
</script>
