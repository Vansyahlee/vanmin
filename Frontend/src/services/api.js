import axios from 'axios'

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('fas_token')

  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }

  return config
})

export default api
// Account changes invalidate old sessions on the server.
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (
      error.response?.status === 401 &&
      !error.config?.url?.includes('/auth/login')
    ) {
      localStorage.removeItem('fas_token')
      localStorage.removeItem('fas_user')
      if (window.location.pathname !== '/login')
        window.location.replace('/login')
    }
    return Promise.reject(error)
  },
)
