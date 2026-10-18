import api from '@/services/api'

export const getProducts = async () => {
    const response = await api.get('/products')
    return response.data
}

export const getProductByID = async (id) => {
    const response = await api.get(`/products/${id}`)
    return response.data
}

export const createProduct = async (data) => {
    const response = await api.post('/products', data)
    return response.data
}

export const updateProduct = async (id, data) => {
    const response = await api.patch(`/products/${id}`, data)
    return response.data
}

export const deactivateProduct = async (id) => {
    const response = await api.delete(`/products/${id}`)
    return response.data
}