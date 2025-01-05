export const calculateStrength = (password?: string): number => {
  if (!password) {
    return 0
  }
  let strength = 0;
  if (password.length >= 8) strength += 20;
  if (/[A-Z]/.test(password)) strength += 20;
  if (/[a-z]/.test(password)) strength += 20;
  if (/[0-9]/.test(password)) strength += 20;
  if (/[!@#$%^&*]/.test(password)) strength += 20;
  return strength;
}

export const getStrengthLabel = (strength: number): string => {
  if (strength === 0) return "Very Weak";
  if (strength <= 20) return "Weak";
  if (strength <= 40) return "Fair";
  if (strength <= 60) return "Good";
  if (strength <= 80) return "Strong";
  return "Very Strong";
}

export const getStrengthColor = (strength: number): string => {
  if (strength <= 20) return "bg-red-500";
  if (strength <= 40) return "bg-orange-500";
  if (strength <= 60) return "bg-yellow-500";
  if (strength <= 80) return "bg-lime-500";
  return "bg-green-500";
}


export const validateEmail = (email?: string):boolean =>{
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return email? emailRegex.test(email):false
}
