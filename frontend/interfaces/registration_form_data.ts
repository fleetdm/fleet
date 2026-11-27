export interface IRegistrationFormData {
  name: string;
  password: string;
  password_confirmation: string;
  email: string;
  org_name: string;
  org_web_url: string;
  org_logo_file?: File | null;
  fleet_web_address: string;
  server_url: string;
}
