provider "cleaneks" {
  host                   = data.aws_eks_cluster.cluster.endpoint
  cluster_ca_certificate = base64decode(data.aws_eks_cluster.cluster.certificate_authority[0].data)
  token                  = data.aws_eks_cluster_auth.cluster.token
}

module "eks" {
  source = "terraform-aws-modules/eks/aws"

  enable_cluster_creator_admin_permissions = true # if this is disabled then the deployment user cannot work inside kubernetes cluster
}

data "aws_eks_cluster" "cluster" {
  name       = module.eks.cluster_name
  depends_on = [module.eks]
}

data "aws_eks_cluster_auth" "cluster" {
  name       = module.eks.cluster_name
  depends_on = [module.eks]
}
