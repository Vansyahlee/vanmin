import api from '@/services/api'

export const getUsers = async (params) =>
  (await api.get('/users', { params })).data
export const createUser = async (payload) =>
  (await api.post('/users', payload)).data
export const updateUser = async (id, payload) =>
  (await api.patch(`/users/${id}`, payload)).data
export const resetUserPassword = async (id, payload) =>
  (await api.post(`/users/${id}/password`, payload)).data
export const getUserHistory = async (id) =>
  (await api.get(`/users/${id}/history`)).data
