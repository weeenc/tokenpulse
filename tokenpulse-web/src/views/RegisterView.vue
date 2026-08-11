<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import AuthShell from '../components/AuthShell.vue';
import { errorMessage } from '../api/client.js';
import { useAuthStore } from '../stores/auth.js';
const form = reactive({ username: '', email: '', password: '' });
const loading = ref(false);
const auth = useAuthStore();
const route = useRoute();
const router = useRouter();
async function submit() {
  loading.value = true;
  try {
    await auth.register(form.username, form.email, form.password);
    await router.push(
      typeof route.query.returnUrl === 'string' ? route.query.returnUrl : '/dashboard',
    );
  } catch (error) {
    ElMessage.error(errorMessage(error));
  } finally {
    loading.value = false;
  }
}
</script>
<template>
  <AuthShell title="创建账户" subtitle="一个账户，连接你的每一台开发设备。"
    ><el-form label-position="top" @submit.prevent="submit"
      ><el-form-item label="用户名"
        ><el-input v-model="form.username" size="large" autocomplete="username" /></el-form-item
      ><el-form-item label="邮箱（可选）"
        ><el-input v-model="form.email" size="large" autocomplete="email" /></el-form-item
      ><el-form-item label="密码"
        ><el-input
          v-model="form.password"
          size="large"
          type="password"
          show-password
          autocomplete="new-password" /></el-form-item
      ><el-button
        type="primary"
        size="large"
        native-type="submit"
        :loading="loading"
        class="full-button"
        >注册并继续</el-button
      ></el-form
    >
    <p class="auth-switch">
      已有账户？<router-link :to="{ path: '/login', query: route.query }">直接登录</router-link>
    </p></AuthShell
  >
</template>
