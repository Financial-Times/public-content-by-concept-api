class public_content_by_concept_api {

  $configParameters = hiera('configParameters','')

  class { "go_service_profile" :
    service_module => $module_name,
    service_name => 'public-content-by-concept-api',
    configParameters => $configParameters
  }

}
