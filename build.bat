cd C:\Data\terraform-provider-cleaneks
del %appdata%\terraform.d\plugins\registry.terraform.io\taliesins\cleaneks\1.0.3\windows_amd64\terraform-provider-cleaneks_1.0.3.exe
go build -o %appdata%\terraform.d\plugins\registry.terraform.io\taliesins\cleaneks\1.0.3\windows_amd64\terraform-provider-cleaneks_1.0.3.exe
cd C:\Data\terraform-provider-cleaneks\examples\resources\cleaneks_iso_image
del .terraform.lock.hcl /Q
RMDIR .terraform /S /Q
#set TF_LOG=TRACE
#set TF_LOG_CORE=TRACE
set TF_LOG_PROVIDER=TRACE
terraform init