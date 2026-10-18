import api from '@/services/api'

export const getCustomers = async () => {
    const response = await api.get(
        '/customers',
    )

    return response.data
}

export const getCustomerByID = async id => {
    const response = await api.get(
        `/customers/${id}`,
    )

    return response.data
}

export const createCustomer = async data => {
    const response = await api.post(
        '/customers',
        data,
    )

    return response.data
}

export const updateCustomer = async (
    id,
    data,
) => {
    const response = await api.patch(
        `/customers/${id}`,
        data,
    )

    return response.data
}

export const deleteCustomer = async id => {
    const response = await api.delete(
        `/customers/${id}`,
    )

    return response.data
}